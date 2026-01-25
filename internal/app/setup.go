package app

import (
	"errors"

	"github.com/dnstapir/tapir-analyse-new-qname/internal/common"
)

type Conf struct {
	Log       common.Logger
	Nats      nats
	Tapir     tapir
	Validator validator
}

type nats interface {
	Publish(string) error
	Subscribe() (<-chan common.NatsMsg, error)
	CheckDomain(fqdn string) (bool, int64)
	StoreNewDomain(fqdn string, thumbprint string) error
	RefreshNewQnameEvent(fqdn string, thumbprint string) error
}

type tapir interface {
	GenerateObservationMsg(domain string, flags uint32) (string, error)
	ExtractDomain(msgJson string) (string, error)
}

type validator interface {
	Validate(data []byte) (bool, string)
}

func Create(conf Conf) (*application, error) {
	a := new(application)

	a.log = conf.Log
	if a.log == nil {
		return nil, errors.New("nil logger")
	}

	a.n = conf.Nats
	if a.n == nil {
		return nil, errors.New("nil nats")
	}

	a.t = conf.Tapir
	if a.t == nil {
		return nil, errors.New("nil tapir")
	}

	a.v = conf.Validator
	if a.v == nil {
		return nil, errors.New("nil validator")
	}

	return a, nil
}
