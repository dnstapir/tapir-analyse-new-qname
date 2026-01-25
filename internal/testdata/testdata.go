package testdata

import (
	"embed"
	"path"
	"path/filepath"
	"runtime"
)

//go:embed messages
var Msgs embed.FS

//go:embed messages/bad/dummy.json
var MsgDummy string

//go:embed messages/bad/empty.json
var MsgEmpty string

//go:embed messages/new_qname/schema90202b31/basic.json
var MsgNewQname90202b31Basic string

//go:embed messages/observations/schema90202b31/basic.json
var MsgObservations90202b31Basic string

/*
 * github.com/santhosh-tekuri/jsonschema/v6 cannot simply load files
 * from an embed.FS. Hence, we store the absolute paths to schema files
 * we might want to test as strings. They can then be used by importing
 * this testdata package and invoking e.g. "Compile(testdata.SchemaAcceptAll)"
 */
var _, testdataFile, _, _ = runtime.Caller(0)
var testdataDir = path.Dir(testdataFile)
var SchemaDir, _ = filepath.Abs(filepath.Join(testdataDir, "schemas"))
var SchemaObservations90202b31, _ = filepath.Abs(filepath.Join(testdataDir, "schemas/observations_90202b31.json"))
var SchemaNewQname90202b31, _ = filepath.Abs(filepath.Join(testdataDir, "schemas/new_qname_90202b31.json"))
