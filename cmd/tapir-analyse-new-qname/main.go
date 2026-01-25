package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pelletier/go-toml/v2"

	"github.com/dnstapir/tapir-analyse-new-qname/internal/app"
	"github.com/dnstapir/tapir-analyse-new-qname/internal/libtapir"
	"github.com/dnstapir/tapir-analyse-new-qname/internal/logger"
	"github.com/dnstapir/tapir-analyse-new-qname/internal/nats"
	"github.com/dnstapir/tapir-analyse-new-qname/internal/schemaval"
)

/* Rewritten if building with make */
var version = "BAD-BUILD"
var commit = "BAD-BUILD"

const env_TAPIR_ANALYSE_NATS_URL = "TAPIR_ANALYSE_NATS_URL"

type mainConf struct {
	Nats      natsConf `toml:"nats"`
	SchemaDir string   `toml:"schema_dir"`
}

type natsConf struct {
	Url        string `toml:"url"`
	InSubject  string `toml:"in_subject"`
	OutSubject string `toml:"out_subject"`
}

func main() {
	var configFile string
	var runVersionCmd bool
	var debugFlag bool
	var conf mainConf

	flag.BoolVar(&runVersionCmd,
		"version",
		false,
		"Print version then exit",
	)
	flag.StringVar(&configFile,
		"config-file",
		"/etc/dnstapir/tapir-analyse-new-qname.toml",
		"Configuration file to use",
	)
	flag.BoolVar(&debugFlag,
		"debug",
		false,
		"Enable DEBUG logs",
	)
	flag.Parse()

	log, err := logger.Create(
		logger.Conf{
			Debug: debugFlag,
		})
	if err != nil {
		panic(fmt.Sprintf("Could not create logger, err: '%s'", err))
	}

	log.Info("tapir-analyse-new-qname version %s-%s", version, commit)

	if runVersionCmd {
		/* We've just printed the version, we are done */
		os.Exit(0)
	}

	log.Debug("Debug logging enabled")

	file, err := os.Open(configFile)
	if err != nil {
		log.Error("Couldn't open config file '%s', exiting...\n", configFile)
		os.Exit(-1)
	}

	confDecoder := toml.NewDecoder(file)
	if confDecoder == nil {
		log.Error("Problem decoding config file '%s', exiting...\n", configFile)
		os.Exit(-1)
	}

	confDecoder.DisallowUnknownFields()
	err = confDecoder.Decode(&conf)
	if err != nil {
		log.Error("Problem decoding config file: '%s', exiting...\n", err)
		os.Exit(-1)
	}

	/* If set, environment variables override config file */
	envNatsURL, exists := os.LookupEnv(env_TAPIR_ANALYSE_NATS_URL)
	if exists && envNatsURL != "" {
		log.Info("Envvar \"%s\" is set, will use it to connect to NATS", envNatsURL)
		conf.Nats.Url = envNatsURL
	}

	nConf := nats.Conf{
		Log:        log,
		Url:        conf.Nats.Url,
		InSubject:  conf.Nats.InSubject,
		OutSubject: conf.Nats.OutSubject,
	}
	natsClient, err := nats.Create(nConf)
	if err != nil {
		log.Error("Error creating NATS client: '%s', exiting...\n", err)
		os.Exit(-1)
	}

	vConf := schemaval.Conf{
		SchemaDir: conf.SchemaDir,
		Log:       log,
	}
	val, err := schemaval.Create(vConf)
	if err != nil {
		log.Error("Error creating schema validator: '%s', exiting...\n", err)
		os.Exit(-1)
	}

	tapir, err := libtapir.Create(libtapir.Conf{})
	if err != nil {
		log.Error("Error creating tapir library handle: '%s', exiting...\n", err)
		os.Exit(-1)
	}

	appConf := app.Conf{
		Log:       log,
		Nats:      natsClient,
		Validator: val,
		Tapir:     tapir,
	}

	application, err := app.Create(appConf)
	if err != nil {
		log.Error("Error building application: '%s', exiting...\n", err)
		os.Exit(-1)
	}

	sigChan := make(chan os.Signal, 1)
	defer close(sigChan)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	done := application.Run()

	select {
	case s := <-sigChan:
		log.Info("Got signal '%s', exiting...\n", s)
	case err := <-done:
		if err != nil {
			log.Error("App exited with error: '%s'\n", err)
		} else {
			log.Info("Done!\n")
		}
	}

	err = application.Stop()
	if err != nil {
		log.Error("Error stopping app: '%s'\n", err)
		os.Exit(-1)
	}

	os.Exit(0)
}
