package nats

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/dnstapir/tapir-analyse-new-qname/internal/common"
)

// TODO find more suitable location to put the analyst identifier
const analystNatsIdentifier string = "tapir-analyse-new-qname"

type Conf struct {
	Debug                    bool   `toml:"debug"`
	Url                      string `toml:"url"`
	EventSubject             string `toml:"event_subject"`
	ObservationSubjectPrefix string `toml:"observation_subject_prefix"`
	PrivateSubjectPrefix     string `toml:"private_subject_prefix"`
	SeenDomainsSubjectPrefix string `toml:"seen_domains_subject_prefix"`
	GloballyNewBucket        string `toml:"globally_new_bucket"`
	PrivateBucket            string `toml:"private_bucket"`
	SeenDomainsBucket        string `toml:"seen_domains_bucket"`
	//MultiNewLimit            string `toml:"multi_new_limit"`
	//MultiNewThreshold        string `toml:"multi_new_threshold"`
	Log common.Logger
}

type natsClient struct {
	log                      common.Logger
	url                      string
	eventSubject             string
	observationSubjectPrefix string
	privateSubjectPrefix     string
	seenDomainsSubjectPrefix string
	globallyNewBucket        string
	privateBucket            string
	seenDomainsBucket        string
	kvGloballyNew            jetstream.KeyValue
	kvPrivate                jetstream.KeyValue
	kvSeenDomains            jetstream.KeyValue
	conn                     *nats.Conn
}

func Create(conf Conf) (*natsClient, error) {
	nc := new(natsClient)

	if conf.Log == nil {
		return nil, errors.New("nil logger")
	}
	nc.log = conf.Log

	if conf.Url == "" {
		return nil, errors.New("no NATS URL")
	}
	nc.url = conf.Url

	if conf.EventSubject == "" {
		return nil, errors.New("no event subject")
	}
	nc.eventSubject = strings.Trim(conf.EventSubject, common.NATS_DELIM)

	if conf.ObservationSubjectPrefix == "" {
		return nil, errors.New("no observation subject prefix")
	}
	nc.observationSubjectPrefix = strings.Trim(conf.ObservationSubjectPrefix, common.NATS_DELIM)

	if conf.PrivateSubjectPrefix == "" {
		return nil, errors.New("no private subject prefix")
	}
	nc.privateSubjectPrefix = strings.Trim(conf.PrivateSubjectPrefix, common.NATS_DELIM)

	if conf.SeenDomainsSubjectPrefix == "" {
		return nil, errors.New("no seen domains subject prefix")
	}
	nc.seenDomainsSubjectPrefix = strings.Trim(conf.SeenDomainsSubjectPrefix, common.NATS_DELIM)

	if conf.GloballyNewBucket == "" {
		return nil, errors.New("no globally_new bucket")
	}
	nc.globallyNewBucket = conf.GloballyNewBucket

	if conf.PrivateBucket == "" {
		return nil, errors.New("no private bucket")
	}
	nc.privateBucket = conf.PrivateBucket

	if conf.SeenDomainsBucket == "" {
		return nil, errors.New("no seen domains bucket")
	}
	nc.seenDomainsBucket = conf.SeenDomainsBucket

	err := nc.initNats()
	if err != nil {
		nc.log.Error("Error initializing NATS")
		return nil, err
	}

	nc.log.Debug("NATS debug logging enabled")
	return nc, nil
}

func (nc *natsClient) ActivateSubscription(ctx context.Context) (<-chan common.NatsMsg, error) {
	rawChan := make(chan *nats.Msg, 100) // TODO adjustable buffer?
	sub, err := nc.conn.ChanSubscribe(nc.eventSubject, rawChan)
	if err != nil {
		nc.log.Error("Couldn't subscribe to raw nats channel: '%s'", err)
		return nil, err
	}

	outCh := make(chan common.NatsMsg, 100) // TODO adjustable buffer?
	go func() {
		defer close(outCh)
		defer func() { _ = sub.Unsubscribe() }()
		nc.log.Info("Starting NATS listener loop")
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-rawChan:
				if !ok {
					nc.log.Warning("Incoming NATS channel closed")
					return
				}
				nc.log.Debug("Incoming NATS message on '%s'!", msg.Subject)
				msg.Ack()
				natsMsg := common.NatsMsg{
					Headers: make(map[string]string),
					Data:    msg.Data,
					Subject: msg.Subject,
				}
				for h, v := range msg.Header {
					if slices.Contains(common.NATSHEADERS_DNSTAPIR_ALL, h) {
						natsMsg.Headers[h] = v[0] // TODO use entire slice?
					}
				}
				select {
				case outCh <- natsMsg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	nc.log.Info("Subscribed to '%s'", nc.eventSubject)

	return outCh, nil
}

func (nc *natsClient) SetObservationGloballyNew(ctx context.Context, domain string) error {
	flipped := _flipDomainName(domain)
	subject := strings.Join(
		[]string{
			nc.observationSubjectPrefix,
			common.OBS_GLOBALLY_NEW,
			flipped,
		},
		common.NATS_DELIM)

	entry, err := nc.kvGloballyNew.Get(ctx, subject)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		_, err := nc.kvGloballyNew.Put(ctx, subject, []byte(analystNatsIdentifier))
		if err != nil {
			nc.log.Error("Couldn't set key '%s': '%s'", subject, err)
			return err
		}
	} else if err == nil {
		_, err := nc.kvGloballyNew.Update(ctx, subject, []byte(analystNatsIdentifier), entry.Revision())
		if err != nil {
			nc.log.Error("Couldn't update report history: %s", err)
			return err
		}
	} else {
		nc.log.Error("Couldn't get key '%s': '%s'", subject, err)
		return err
	}

	return nil
}

func (nc *natsClient) AddDomain(ctx context.Context, domain string, reporter string) (bool, error) {
	subject := getSubjectFromFqdn(nc.seenDomainsSubjectPrefix, domain, "")

	entry, err := nc.kvSeenDomains.Get(ctx, subject)
	if err != nil && !errors.Is(err, jetstream.ErrKeyNotFound) {
		nc.log.Error("Error accessing storage: %s, subject: %s", err, subject)
		return false, err
	}

	var updatedData []byte
	found := false
	timestamp := time.Now().Unix()
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		nc.log.Debug("Previously unseen domain '%s' by '%s'", domain, reporter)
		firstReport := make(map[string]int64)
		firstReport[reporter] = timestamp
		updatedData, err = json.Marshal(firstReport)
		if err != nil {
			nc.log.Warning("Couldn't serialize first report: %s", err)
			return found, nil
		}
		_, err = nc.kvSeenDomains.Put(ctx, subject, updatedData)
		if err != nil {
			nc.log.Warning("Couldn't update report history: %s", err)
		}
	} else if err == nil {
		found = true
		data := entry.Value()
		var reportHistory map[string]int64
		err = json.Unmarshal(data, &reportHistory)
		if err != nil {
			nc.log.Warning("Couldn't read report history: %s", err)
			return found, nil
		}

		_, ok := reportHistory[reporter]
		if !ok {
			reportHistory[reporter] = timestamp
		} else {
			nc.log.Debug("%s has already reported %s as seen", reporter, domain)
		}
		updatedData, err = json.Marshal(reportHistory)
		if err != nil {
			nc.log.Warning("Couldn't serialize updated report: %s", err)
			return found, nil
		}
		_, err := nc.kvSeenDomains.Update(ctx, subject, updatedData, entry.Revision())
		if err != nil {
			nc.log.Error("Couldn't update report history: %s", err)
			return found, err
		}
	} else {
		panic("unreachable")
	}

	return found, nil
}

func (nc *natsClient) Shutdown() error {
	// TODO impl
	return nil
}

func (nc *natsClient) initNats() error {
	conn, err := nats.Connect(nc.url)
	if err != nil {
		nc.log.Error("Error connecting to nats while setting up KV store: %s", err)
		return err
	}
	js, err := jetstream.New(conn)
	if err != nil {
		nc.log.Error("Error creating jetstream handle: %s", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	kvGloballyNew, err := js.KeyValue(ctx, nc.globallyNewBucket)
	if err != nil {
		nc.log.Error("Error looking up key value store in NATS: %s", err)
		return err
	}

	kvPrivate, err := js.CreateKeyValue(ctx,
		jetstream.KeyValueConfig{
			Bucket:         nc.privateBucket,
			TTL:            time.Duration(20) * time.Second, // TODO use MultiNewLimit
			LimitMarkerTTL: time.Duration(0) * time.Second,
		})
	if err != nil {
		nc.log.Error("Error creating private key-value store in NATS: %s", err)
		return err
	}

	kvSeenDomains, err := js.CreateKeyValue(ctx, // TODO Should this bucket be provisioned by this analyst?
		jetstream.KeyValueConfig{
			Bucket: nc.seenDomainsBucket,
		})
	if err != nil {
		nc.log.Error("Error creating seen domains key-value store in NATS: %s", err)
		return err
	}

	nc.kvGloballyNew = kvGloballyNew
	nc.kvPrivate = kvPrivate
	nc.kvSeenDomains = kvSeenDomains
	nc.conn = conn
	nc.log.Debug("Nats key value store created successfully!")

	return nil
}

func _flipDomainName(domain string) string {
	split := strings.Split(strings.Trim(domain, common.NATS_DELIM), common.NATS_DELIM)
	slices.Reverse(split)
	return strings.Join(split, common.NATS_DELIM)
}
