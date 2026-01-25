package app

import (
	"testing"

	"github.com/dnstapir/tapir-analyse-new-qname/internal/common"
	"github.com/dnstapir/tapir-analyse-new-qname/internal/libtapir"
	"github.com/dnstapir/tapir-analyse-new-qname/internal/logger"
	"github.com/dnstapir/tapir-analyse-new-qname/internal/schemaval"
	"github.com/dnstapir/tapir-analyse-new-qname/internal/testdata"
)

func TestAppDummy(t *testing.T) {
	log, err := logger.Create(
		logger.Conf{
			Debug: true,
		})
	if err != nil {
		t.Fatalf("Could not create logger, err: '%s'", err)
	}
	nats := new(fakeNats)

	tapir, err := libtapir.Create(libtapir.Conf{})
	if err != nil {
		t.Fatalf("Error creating tapir handle: %s", err)
	}

	vConf := schemaval.Conf{
		SchemaDir: testdata.SchemaDir,
		Log:       log,
	}
	val, err := schemaval.Create(vConf)
	if err != nil {
		t.Fatalf("Error creating schema validator: '%s', exiting...\n", err)
	}

	appConf := Conf{
		Log:       log,
		Nats:      nats,
		Tapir:     tapir,
		Validator: val,
	}

	application, err := Create(appConf)
	if err != nil {
		t.Fatalf("Error creating app: %s", err)
	}

	natsMsg := common.NatsMsg{
		Data:    []byte(testdata.MsgNewQname90202b31Basic),
		Headers: map[string]string{common.NATSHEADER_KEY_THUMBPRINT: "test-thumbprint"},
	}
	application.handleMsg(natsMsg)

	got := nats.latestPublished

	if !val.ValidateWithID([]byte(got), "https://schema.dnstapir.se/v1/core_observation") {
		t.Fatalf("Validation failed for %s", got)
	}

	if len(nats.domainStorage) != 1 {
		t.Fatalf("unexpected number of entries in domain storage. Got: %d, expected: %d", len(nats.domainStorage), 1)
	}
}
