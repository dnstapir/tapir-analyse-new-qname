package libtapir

import (
	"fmt"
	"testing"

	"github.com/dnstapir/tapir-analyse-new-qname/internal/logger"
	"github.com/dnstapir/tapir-analyse-new-qname/internal/schemaval"
	"github.com/dnstapir/tapir-analyse-new-qname/internal/testdata"
)

func TestGenerateObservationMsg(t *testing.T) {
	tapirHandle, err := Create(Conf{})
	if err != nil {
		t.Fatalf("Error creating tapir handle: %s", err)
	}

	log, err := logger.Create(
		logger.Conf{
			Debug: true,
		})
	if err != nil {
		t.Fatalf("Could not create logger, err: '%s'", err)
	}

	conf := schemaval.Conf{
		SchemaDir: testdata.SchemaDir,
		Log:       log,
	}

	sv, err := schemaval.Create(conf)
	if err != nil {
		t.Fatalf("Could not create schemaval obj, err: '%s'", err)
	}

	msg, err := tapirHandle.GenerateObservationMsg("lala.xa", 1)
	ok := sv.ValidateWithID([]byte(msg), "https://schema.dnstapir.se/v1/core_observation")
	if !ok {
		t.Fatalf("Generated message does not conform to schema!")
	}
}

func TestExtractDomain(t *testing.T) {
	tapirHandle, err := Create(Conf{})
	if err != nil {
		t.Fatalf("Error creating tapir handle: %s", err)
	}

	msgFmt := `
    {
        "flags": 0,
        "initiator": "test",
        "qclass": 0,
        "qname": "%s",
        "qtype": 0,
        "rdlength": 0,
        "timestamp": "1985-04-12T23:20:50.52Z",
        "type": "test",
        "version": 0
    }`

	wanted := "wanted.xa"
	msg := fmt.Sprintf(msgFmt, wanted)

	domain, err := tapirHandle.ExtractDomain(msg)

	if err != nil {
		t.Fatalf("Error extracting domain: %s", err)
	}

	if domain != wanted {
		t.Fatalf("Error expected: %s, got: %s", wanted, domain)
	}
}
