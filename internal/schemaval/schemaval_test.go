package schemaval

import (
	"path/filepath"
	"testing"

	"github.com/dnstapir/tapir-analyse-new-qname/internal/logger"
	"github.com/dnstapir/tapir-analyse-new-qname/internal/testdata"
)

func setup(t *testing.T) *schemaval {
	t.Helper()
	log, err := logger.Create(
		logger.Conf{
			Debug: true,
		})
	if err != nil {
		t.Fatalf("Could not create logger, err: '%s'", err)
	}

	conf := Conf{
		SchemaDir: testdata.SchemaDir,
		Log:       log,
	}

	sv, err := Create(conf)
	if err != nil {
		t.Fatalf("Could not create schemaval obj, err: '%s'", err)
	}

	return sv
}

func TestSchemaval(t *testing.T) {
	sv := setup(t)

	test := func(t *testing.T, schema string, msgFile string, expected bool) {
		t.Helper()

		t.Run(filepath.Join(schema, msgFile), func(t *testing.T) {
			t.Helper()

			data, err := testdata.Msgs.ReadFile(msgFile)
			if err != nil {
				t.Fatalf("Error opening msg file '%s'", msgFile)
			}

			got := sv.ValidateWithID(data, schema)
			if got != expected {
				t.Fatalf("Unexpected validation outcome. Got: %t, Expected: %t", got, expected)
			}
		})
	}

	test(t,
		"https://schema.dnstapir.se/bad/id",
		"messages/bad/dummy.json",
		false,
	)

	test(t,
		"https://schema.dnstapir.se/v1/new_qname",
		"messages/bad/dummy.json",
		false,
	)
	test(t,
		"https://schema.dnstapir.se/v1/new_qname",
		"messages/bad/empty.json",
		false,
	)
	test(t,
		"https://schema.dnstapir.se/v1/new_qname",
		"messages/new_qname/schema90202b31/basic.json",
		true,
	)
	test(t,
		"https://schema.dnstapir.se/v1/new_qname",
		"messages/observations/schema90202b31/basic.json",
		false,
	)

	test(t,
		"https://schema.dnstapir.se/v1/core_observation",
		"messages/bad/dummy.json",
		false,
	)
	test(t,
		"https://schema.dnstapir.se/v1/core_observation",
		"messages/bad/empty.json",
		false,
	)
	test(t,
		"https://schema.dnstapir.se/v1/core_observation",
		"messages/new_qname/schema90202b31/basic.json",
		false,
	)
	test(t,
		"https://schema.dnstapir.se/v1/core_observation",
		"messages/observations/schema90202b31/basic.json",
		true,
	)
}
