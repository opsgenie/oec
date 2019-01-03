package main

import (
	"flag"
	"fmt"
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/queue"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var addr = flag.String("marid-metrics", "8081", "The address to listen on for HTTP requests.")
var logPath = strings.Join([]string{"opsgenie", "logs", "marid.log"}, string(os.PathSeparator))

var MaridVersion string
var MaridCommitVersion string

func main() {

	flag.Parse()
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		logrus.Infof("Marid-metrics serves in http://localhost:%s/metrics.", *addr)
		logrus.Error("Marid-metrics error: ", http.ListenAndServe(":" + *addr, nil))
	}()

	logrus.SetFormatter(
		&logrus.TextFormatter {
			ForceColors: true,
			FullTimestamp: true,
			TimestampFormat: time.RFC3339Nano,
		},
	)

	logrus.Infof("Marid version is %s", MaridVersion)
	logrus.Infof("Marid commit version is %s", MaridCommitVersion)

	usr, err := user.Current()
	if err != nil {
		logrus.Fatalln(err)
	}

	logger := &lumberjack.Logger {
		Filename:  strings.Join([]string{usr.HomeDir, logPath}, string(os.PathSeparator)),
		MaxSize:   3, 	// MB
		MaxAge:    10, 	// Days
		LocalTime: true,
	}

	logrus.SetOutput(io.MultiWriter(os.Stdout, logger))

	configuration, err := conf.ReadConfFile()
	if err != nil {
		logrus.Fatalln("Could not read configuration: ", err)
	}

	logrus.SetLevel(configuration.LogLevel)

	queueProcessor := queue.NewQueueProcessor(configuration)
	queue.UserAgentHeader = fmt.Sprintf("%s/%s %s (%s/%s)",  MaridVersion, MaridCommitVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH)

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
