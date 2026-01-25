package app

import (
	"errors"

	"github.com/dnstapir/tapir-analyse-new-qname/internal/common"
)

var c_OBS_GLOBAL_NEW uint32 = 2048

type application struct {
	log common.Logger
	n   nats
	t   tapir
	v   validator

	isRunning bool
	doneChan  chan error
	stopChan  chan bool
}

func (a *application) Run() <-chan error {
	a.doneChan = make(chan error, 10)
	a.stopChan = make(chan bool, 1)
	a.isRunning = true

	msgChan, err := a.n.Subscribe()
	if err != nil {
		a.doneChan <- errors.New("error activating nats subscription")
		a.isRunning = false
	}
	if msgChan == nil {
		a.doneChan <- errors.New("msg chan error")
		a.isRunning = false
	}

	go func() {
		for {
			select {
			case msg := <-msgChan:
				a.handleMsg(msg)
			case <-a.stopChan:
				a.log.Info("Stopping main worker thread")
				return
			}
		}
	}()

	return a.doneChan
}

func (a *application) Stop() error {
	if a.log != nil {
		if a.isRunning {
			a.log.Info("Stopping application")
		} else {
			a.log.Info("Stop() called but application was not running")
		}
	}

	a.stopChan <- true

	close(a.doneChan)
	close(a.stopChan)

	return nil
}

func (a *application) handleMsg(msg common.NatsMsg) {
	a.log.Debug("Received message, payload size: %d bytes, # of headers: %d", len(msg.Data), len(msg.Headers))
	defer a.log.Debug("Done handling message")
	if len(msg.Data) == 0 {
		a.log.Debug("Msg had no data, probably garbage. Won't handle...")
		return
	}

	ok, _ := a.v.Validate(msg.Data)
	if !ok {
		a.log.Warning("Invalid message")
		return
	}

	msgDomain, err := a.t.ExtractDomain(string(msg.Data))
	if err != nil {
		a.log.Error("Error reading domain from message: %s", err)
		return
	}
	a.log.Debug("Read domain %s from message", msgDomain)

	exists, _ := a.n.CheckDomain(msgDomain)

	thumbprint, ok := msg.Headers[common.NATSHEADER_KEY_THUMBPRINT]
	if !ok {
		a.log.Error("Missing thumbprint for NEW_QNAME event")
		return
	}

	if exists {
		err = a.n.RefreshNewQnameEvent(msgDomain, thumbprint)
		if err != nil {
			a.log.Error("Error refreshing TTL: %s", err)
		} else {
			a.log.Debug("Handled event for existing domain '%s'", msgDomain)
		}
	} else {
		a.log.Info("Got event for unseen domain '%s'", msgDomain)

		err = a.n.StoreNewDomain(msgDomain, thumbprint)
		if err != nil {
			a.log.Error("Error storing new domain %s in NATS: %s", msgDomain, err)
			return
		}

		outMsg, err := a.t.GenerateObservationMsg(msgDomain, c_OBS_GLOBAL_NEW)
		if err != nil {
			a.log.Error("Error generating message: %s", err)
			return
		}

		err = a.n.Publish(string(outMsg))
		if err != nil {
			a.log.Error("Error publishing nats message!")
		} else {
			a.log.Debug("Published message: %s", string(outMsg))
		}
	}
}
