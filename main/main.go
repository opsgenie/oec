package main

import (
	"flag"
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/queue"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"syscall"
	"time"
)

var newQueueProcessorFunc = queue.NewQueueProcessor
var readConfFileFunc = conf.ReadConfFile

var addr = flag.String("marid-metrics", ":8081", "The address to listen on for HTTP requests.")
const logPath = string(os.PathSeparator) + ".opsgenie" + string(os.PathSeparator) + "marid.log"

func main() {

	flag.Parse()
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(*addr, nil))
	}()

	logrus.SetFormatter(
		&logrus.TextFormatter {
			ForceColors: true,
			FullTimestamp: true,
			TimestampFormat: time.RFC3339Nano,
		},
	)

	usr, err := user.Current()
	if err != nil {
		logrus.Fatalln(err)
	}

	logger := &lumberjack.Logger {
		Filename:  usr.HomeDir + logPath,
		MaxSize:   1, 	// MB
		MaxAge:    10, 	// Days
		LocalTime: true,
	}

	logrus.SetOutput(io.MultiWriter(os.Stdout, logger))

	configuration, err := readConfFileFunc()
	if err != nil {
		logrus.Fatalln("Could not read configuration: ", err)
	}

	logrus.SetLevel(configuration.LogLevel)

	queueProcessor := newQueueProcessorFunc(configuration)

	go func() {
		err = queueProcessor.StartProcessing()
		if err != nil {
			logrus.Fatalln(err)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <- signals:
		logrus.Infof("Marid will be stopped gracefully.")
		err := queueProcessor.StopProcessing()
		if err != nil {
			logrus.Fatalln(err)
		}
	}

	os.Exit(0)
}
