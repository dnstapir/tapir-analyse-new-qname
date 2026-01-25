package nats

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/dnstapir/tapir-analyse-new-qname/internal/common"
)

const c_SUBJECT_SCRATCHPAD_PREFIX = "scratchpad"  // TODO make configurable
const c_BUCKET_NAME = c_SUBJECT_SCRATCHPAD_PREFIX // TODO make configurable
const c_DEFAULT_TTL = 60                          /* seconds */ // TODO make configurable
const c_LIMITMARKER_TTL = 1                       /* seconds */ // TODO make configurable

type Conf struct {
	Url        string
	OutSubject string
	InSubject  string
	Log        common.Logger
}

type natsClient struct {
	log        common.Logger
	url        string
	outSubject string
	inSubject  string
	queue      string
	conn       *nats.Conn
	kv         jetstream.KeyValue
}

func Create(conf Conf) (*natsClient, error) {
	nc := new(natsClient)

	nc.url = conf.Url // TODO validate

	if conf.Log == nil {
		return nil, errors.New("nil logger")
	}
	nc.log = conf.Log

	if conf.OutSubject == "" || conf.InSubject == "" {
		return nil, errors.New("bad nats subject")
	}
	nc.outSubject = conf.OutSubject
	nc.inSubject = conf.InSubject

	err := nc.initNats()
	if err != nil {
		nc.log.Error("Error initializing NATS")
		return nil, err
	}

	return nc, nil
}

func (nc *natsClient) Subscribe() (<-chan common.NatsMsg, error) {
	conn, err := nc.getConn()
	if err != nil {
		return nil, err
	}

	rawCh := make(chan *nats.Msg)
	_, err = conn.ChanSubscribe(nc.inSubject, rawCh)
	if err != nil {
		return nil, err
	}

	outCh := make(chan common.NatsMsg)
	go func() {
		for msg := range rawCh {
			natsMsg := common.NatsMsg{
				Headers: make(map[string]string),
				Data:    msg.Data,
			}
			for h, v := range msg.Header {
				if slices.Contains(common.NATSHEADERS_DNSTAPIR_ALL, h) {
					natsMsg.Headers[h] = v[0] // TODO use entire slice?
				}
			}
			outCh <- natsMsg
			msg.Ack()
		}
		close(outCh)
		conn.Close()
	}()

	return outCh, nil
}

func (nc *natsClient) Publish(msg string) error {
	conn, err := nc.getConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	//natsMsg := nats.NewMsg(nc.outSubject)
	//natsMsg.Data = []byte(msg)

	//err = conn.PublishMsg(natsMsg)
	err = conn.Publish(nc.outSubject, []byte(msg))
	if err != nil {
		return err
	} else {
		nc.log.Debug("Successful publish on '%s'", nc.outSubject)
	}

	return nil
}

func (nc *natsClient) getConn() (*nats.Conn, error) {
	if nc.conn == nil {
		/* Get a new connection */
		return nats.Connect(nc.url)
	}

	/* Use an existing connection, if it exists */
	return nc.conn, nil
}

func (nc *natsClient) initNats() error {
	conn, err := nc.getConn()
	if err != nil {
		nc.log.Error("Error connecting to nats while setting up KV store: %s", err)
		return err
	}
	js, _ := jetstream.New(conn)
	ctx := context.Background()

	kv, err := js.CreateKeyValue(ctx,
		jetstream.KeyValueConfig{
			Bucket:         c_BUCKET_NAME,
			LimitMarkerTTL: c_DEFAULT_TTL * time.Second, // TODO what is a good setting?
		})
	if err != nil {
		nc.log.Error("Error creating key value store in NATS: %s", err)
		return err
	}

	nc.kv = kv
	nc.log.Debug("Nats key value store created successfully!")

	return nil
}

func (nc *natsClient) CheckDomain(fqdn string) (bool, int64) {
	ctx := context.Background()

	subject := getSubjectFromFqdn(c_SUBJECT_SCRATCHPAD_PREFIX, fqdn, "")

	_, err := nc.kv.Get(ctx, subject)
	found := false
	var whenAdded int64 = -1

	if err == nil {
		found = true
		nc.log.Debug("Entry for subject '%s' found", subject)
	} else if errors.Is(err, jetstream.ErrKeyNotFound) {
		nc.log.Debug("No entry for subject '%s'", subject)
	} else {
		nc.log.Error("Error accessing storage: %s, subject: %s", err, subject)
	}

	return found, whenAdded
}

func (nc *natsClient) StoreNewDomain(fqdn, thumbprint string) error {
	ctx := context.Background()
	alreadySeen, _ := nc.CheckDomain(fqdn)

	if !alreadySeen {
		/* Store permanently if this was the first time we saw the fqdn */
		subject := getSubjectFromFqdn(c_SUBJECT_SCRATCHPAD_PREFIX, fqdn, "")
		_, err := nc.kv.Put(ctx, subject, []byte(thumbprint))
		if err != nil {
			nc.log.Error("Error storing value for new subject '%s': %s", subject, err)
			return err
		} else {
			nc.log.Debug("Stored info about %s for the first time", fqdn)
		}

		/* Start a timer for the EDGE-specific event */
		err = nc.RefreshNewQnameEvent(fqdn, thumbprint)
		if err != nil {
			nc.log.Error("Error starting expiry timer for %s", getSubjectFromFqdn(c_SUBJECT_SCRATCHPAD_PREFIX, fqdn, thumbprint))
			return err
		}
		// TODO also set "observations.fqdn.globally_new" since this was the
		// first time we saw fqdn
	}

	return nil
}

func (nc *natsClient) RefreshNewQnameEvent(fqdn string, thumbprint string) error {
	ctx := context.Background()

	subject := getSubjectFromFqdn(c_SUBJECT_SCRATCHPAD_PREFIX, fqdn, thumbprint)

	newlySeen := false /* newly seen by EDGE associated with "thumbprint", that is */
	entry, err := nc.kv.Get(ctx, subject)
	if err == nil {
	} else if errors.Is(err, jetstream.ErrKeyNotFound) {
		newlySeen = true
	} else {
		nc.log.Error("Error accessing storage: %s, subject: %s", err, subject)
		return err
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	if newlySeen {
		_, err := nc.kv.Create(
			ctx,
			subject,
			[]byte(timestamp),
			jetstream.KeyTTL(c_DEFAULT_TTL*time.Second))
		if err != nil {
			nc.log.Error("Error creating key: %s. Subject: %s", err, subject)
		}
		return err
	} else {
		_, err := nc.kv.Update(
			ctx,
			subject,
			[]byte(timestamp),
			entry.Revision())
		if err != nil {
			nc.log.Error("Error refreshing TTL: %s. Subject: %s", err, subject)
		} else {
			nc.log.Debug("Refreshed TTL for %s", subject)
		}
		return err
	}

	return nil
}
