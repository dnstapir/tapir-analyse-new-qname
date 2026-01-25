package app

import (
	"time"

	"github.com/dnstapir/tapir-analyse-new-qname/internal/common"
)

type fakeNats struct {
	latestPublished string
	domainStorage   map[string]int64
}

func (fn *fakeNats) Subscribe() (<-chan common.NatsMsg, error) {
	return nil, nil
}

func (fn *fakeNats) Publish(msg string) error {
	fn.latestPublished = msg
	return nil
}

func (fn *fakeNats) CheckDomain(fqdn string) (bool, int64) {
	created, ok := fn.domainStorage[fqdn]
	return ok, created
}

func (fn *fakeNats) StoreNewDomain(fqdn string, thumbprint string) error {
	if fn.domainStorage == nil {
		fn.domainStorage = make(map[string]int64)
	}

	fn.domainStorage[fqdn] = time.Now().Unix()
	return nil
}

func (fn *fakeNats) RefreshNewQnameEvent(fqdn string, thumbprint string) error {
	return nil
}
