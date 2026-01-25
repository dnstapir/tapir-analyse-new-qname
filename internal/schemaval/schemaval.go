package schemaval

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/dnstapir/tapir-analyse-new-qname/internal/common"
)

type Conf struct {
	Log       common.Logger
	SchemaDir string
}

type schemaval struct {
	log         common.Logger
	schemaStore map[string]*jsonschema.Schema
}

func Create(conf Conf) (*schemaval, error) {
	newSchemaval := new(schemaval)

	if conf.Log == nil {
		return nil, errors.New("error setting logger")
	}
	newSchemaval.log = conf.Log

	if conf.SchemaDir == "" {
		return nil, errors.New("no schema directory specified")
	}

	files, err := os.ReadDir(conf.SchemaDir)
	if err != nil {
		newSchemaval.log.Error("Error reading schema dir %s", conf.SchemaDir)
		return nil, err
	}
	if len(files) == 0 {
		newSchemaval.log.Error("No schemas found in %s", conf.SchemaDir)
		return nil, errors.New("no schemas found")
	}

	newSchemaval.schemaStore = make(map[string]*jsonschema.Schema)
	c := jsonschema.NewCompiler()
	for _, file := range files {
		fullName := filepath.Join(conf.SchemaDir, file.Name())
		schema, err := c.Compile(fullName)
		if err != nil {
			newSchemaval.log.Error("Compiling schema %s failed: %s", file, err)
			return nil, err
		}

		newSchemaval.schemaStore[schema.ID] = schema
	}

	return newSchemaval, nil
}

func (s *schemaval) ValidateWithID(data []byte, id string) bool {
	schema, ok := s.schemaStore[id]
	if !ok {
		s.log.Warning("Requested schema %s not found", id)
		return false
	}

	dataReader := bytes.NewReader(data)
	obj, err := jsonschema.UnmarshalJSON(dataReader)
	if err != nil {
        s.log.Warning("Error unmarshalling byte stream into JSON object: %s", err)
		return false
	}

	err = schema.Validate(obj)
	if err != nil {
		s.log.Debug("Validation error '%s'", err)
		return false
	}

	return true
}

func (s *schemaval) Validate(data []byte) (bool, string) {
    matchedID := ""
    ok := false
    for id, _ := range s.schemaStore {
        ok = s.ValidateWithID(data, id)
        if ok {
            s.log.Debug("Validation match with schema: %s", id)
            matchedID = id
            break
        }
    }

	return ok, matchedID
}
