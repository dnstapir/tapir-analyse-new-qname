package libtapir

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/dnstapir/edm/pkg/protocols"

	"github.com/dnstapir/tapir-analyse-new-qname/internal/common"
)

type Conf struct {
	Log   common.Logger
	Debug bool `toml:"debug"`
}

type libtapir struct {
	log common.Logger
}

func Create(conf Conf) (*libtapir, error) {
	lt := new(libtapir)
	if conf.Log == nil {
		return nil, common.ErrBadHandle
	}

	lt.log = conf.Log

	lt.log.Debug("Libtapir debug logging enabled")
	return lt, nil
}

func (lt *libtapir) ExtractDomain(msgJson []byte) (string, error) {
	var newQnameEvent protocols.NewQnameJSON
	dec := json.NewDecoder(bytes.NewReader(msgJson))

	dec.DisallowUnknownFields()

	err := dec.Decode(&newQnameEvent)
	if err != nil {
		lt.log.Error("Error decoding qname from 'new qname event' msg")
		return "", err
	}

	return string(strings.Trim(newQnameEvent.Qname, ".")), nil
}
