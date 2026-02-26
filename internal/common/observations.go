package common

import (
	"time"
)

// TODO observation encoder can iterate over map and create all
// observation buckets. Micro-analysts can just use the one they need.

const OBS_GLOBALLY_NEW string = "globally_new"
const OBS_LOOPTEST string = "looptest"

type Observation struct {
	Encoding uint32
	Bucket   string
	Ttl      time.Duration
}

var OBS_MAP = map[string]Observation{
	OBS_GLOBALLY_NEW: Observation{
		Encoding: 1,
		Bucket:   OBS_GLOBALLY_NEW,
		Ttl:      7200 * time.Second,
	},
	OBS_LOOPTEST: Observation{
		Encoding: 1024,
		Bucket:   OBS_LOOPTEST,
		Ttl:      3600 * time.Second,
	},
}
