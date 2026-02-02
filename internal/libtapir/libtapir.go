package libtapir

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/dnstapir/edm/pkg/protocols"
	"github.com/dnstapir/tapir"

	"github.com/dnstapir/tapir-analyse-new-qname/internal/common"
)

type Conf struct {
	Log common.Logger
}

type libtapir struct {
	log common.Logger
}

func Create(conf Conf) (*libtapir, error) {
	lt := new(libtapir)
	if conf.Log == nil {
		lt.log = common.FakeLogger{}
	} else {
		lt.log = conf.Log
	}

	return lt, nil
}

func (lt *libtapir) GenerateObservationMsg(domainStr string, flags uint32) (string, error) {
	domain := tapir.Domain{
		Name:         domainStr,
		TimeAdded:    time.Now(),
		TTL:          3600,
		TagMask:      tapir.TagMask(flags),
		ExtendedTags: []string{},
	}

	tapirMsg := tapir.TapirMsg{
		SrcName:   "dns-tapir",
		Creator:   "tapir-analyse-new-qname",
		MsgType:   "observation",
		ListType:  "doubtlist",
		Added:     []tapir.Domain{domain},
		Removed:   []tapir.Domain{},
		Msg:       "",
		TimeStamp: time.Now(),
		TimeStr:   "",
	}

	outMsg, err := json.Marshal(tapirMsg)
	if err != nil {
		lt.log.Error("Error serializing message, discarding...")
		return "", err
	}

	return string(outMsg), nil
}

func (lt *libtapir) ExtractDomain(msgJson string) (string, error) {
	var newQnameEvent protocols.NewQnameJSON
	dec := json.NewDecoder(strings.NewReader(msgJson))

	dec.DisallowUnknownFields()

	err := dec.Decode(&newQnameEvent)
	if err != nil {
		lt.log.Error("Error decoding qname from 'new qname event' msg")
		return "", err
	}

	return string(newQnameEvent.Qname), nil
}
