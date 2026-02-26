package app

import (
	"context"
	"strings"
	"sync"

	"github.com/dnstapir/tapir-analyse-new-qname/internal/common"
)

const c_N_HANDLERS = 3

type Conf struct {
	Debug          bool     `toml:"debug"`
	IgnoreSuffixes []string `toml:"ignore_suffixes"`
	Log            common.Logger
	NatsHandle     nats
	LibtapirHandle libtapir
}

type appHandle struct {
	id             string
	ignoreSuffixes []string
	log            common.Logger
	natsHandle     nats
	libtapirHandle libtapir
	exitCh         chan<- common.Exit
	pm
}

type pm struct {
}

type job struct {
	msg common.NatsMsg
}

type nats interface {
	ActivateSubscription(context.Context) (<-chan common.NatsMsg, error)
	SetObservationGloballyNew(context.Context, string) error
	AddDomain(context.Context, string, string) (bool, error)
	Shutdown() error
}

type libtapir interface {
	ExtractDomain([]byte) (string, error)
}

func Create(conf Conf) (*appHandle, error) {
	a := new(appHandle)
	a.id = "main app"

	if conf.Log == nil {
		return nil, common.ErrBadHandle
	}
	a.log = conf.Log

	if conf.NatsHandle == nil {
		return nil, common.ErrBadHandle
	}
	a.natsHandle = conf.NatsHandle

	if conf.LibtapirHandle == nil {
		return nil, common.ErrBadHandle
	}
	a.libtapirHandle = conf.LibtapirHandle

	for _, s := range conf.IgnoreSuffixes {
		a.ignoreSuffixes = append(a.ignoreSuffixes, strings.Trim(s, "."))
	}

	a.log.Debug("Main app debug logging enabled")
	return a, nil
}

func (a *appHandle) Run(ctx context.Context, exitCh chan<- common.Exit) {
	var natsChan <-chan common.NatsMsg
	var err error
	a.id = "main app"
	a.exitCh = exitCh
	jobChan := make(chan job, 10)

	natsChan, err = a.natsHandle.ActivateSubscription(ctx)
	if err != nil {
		a.log.Error("Couldn't activate NATS subscription: '%s'", err)
		a.exitCh <- common.Exit{ID: a.id, Err: err}
		return
	}

	var wg sync.WaitGroup
	for range c_N_HANDLERS {
		wg.Go(func() {
			for j := range jobChan {
				a.handleJob(ctx, j)
			}
			a.log.Info("Worker done!")
		})
	}

MAIN_APP_LOOP:
	for {
		select {
		case natsMsg, ok := <-natsChan:
			if !ok {
				a.log.Warning("NATS channel closed")
				natsChan = nil
			} else {
				a.log.Debug("Incoming NATS message")
				j := job{
					msg: natsMsg,
				}
				jobChan <- j
			}
		case <-ctx.Done():
			a.log.Info("Stopping main worker thread")
			break MAIN_APP_LOOP
		}
	}

	close(jobChan)

	wg.Wait()

	err = a.natsHandle.Shutdown()
	if err != nil {
		a.log.Error("Encountered '%s' during NATS shutdown", err)
	}

	a.exitCh <- common.Exit{ID: a.id, Err: err}
	a.log.Info("Main app shutdown done")
	return
}

func (a *appHandle) handleJob(ctx context.Context, j job) {
	a.handleMsg(ctx, j.msg)
}

func (a *appHandle) handleMsg(ctx context.Context, msg common.NatsMsg) {
	a.log.Debug("Handling %d byte message on subject %s", len(msg.Data), msg.Subject)
	if len(msg.Data) <= 0 {
		a.log.Warning("Msg had no data, probably garbage. Won't handle...")
		return
	}

	// TODO schema validation
	//	ok, _ := a.v.Validate(msg.Data)
	//	if !ok {
	//		a.log.Warning("Invalid message")
	//		return
	//	}
	//

	msgDomain, err := a.libtapirHandle.ExtractDomain(msg.Data)
	if err != nil {
		a.log.Error("Error reading domain from message: %s", err)
		return
	}

	for _, s := range a.ignoreSuffixes {
		if strings.HasSuffix(msgDomain, s) {
			a.log.Debug("%s matches suffix %s, ignoring...", msgDomain, s)
			return
		}
	}

	thumbprint, ok := msg.Headers[common.NATSHEADER_KEY_THUMBPRINT]
	if !ok {
		a.log.Error("Missing thumbprint for NEW_QNAME event, discarding...")
		return
	}

	alreadyExists, err := a.natsHandle.AddDomain(ctx, msgDomain, thumbprint)
	if err != nil {
		a.log.Error("Couldn't update seen domains: %s", err)
		return
	}

	if alreadyExists {
		// TODO multi-new logic here
		a.log.Debug("Handled event for existing domain '%s'", msgDomain)
		return
	} else {
		a.log.Info("Got event for unseen domain '%s'", msgDomain)

		err = a.natsHandle.SetObservationGloballyNew(ctx, msgDomain)
		if err != nil {
			a.log.Error("Error setting globally_new observation for %s in NATS: %s", msgDomain, err)
			return
		}
		a.log.Debug("Handled event for new domain '%s'", msgDomain)
	}
}
