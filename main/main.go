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
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

var metricAddr = flag.String("ois-metrics", "8082", "The address to listen on for HTTP requests.")
var defaultLogFilepath = filepath.Join("/var", "log", "opsgenie", "ois.log")

var OISVersion string
var OISCommitVersion string

func main() {

	logrus.SetFormatter(
		&logrus.TextFormatter{
			ForceColors:     true,
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		},
	)

	logger := &lumberjack.Logger{
		Filename:  defaultLogFilepath,
		MaxSize:   3,  // MB
		MaxAge:    10, // Days
		LocalTime: true,
	}

	logrus.SetOutput(io.MultiWriter(os.Stdout, logger))

	logrus.Infof("OIS version is %s", OISVersion)
	logrus.Infof("OIS commit version is %s", OISCommitVersion)

	configuration, err := conf.ReadConfFile()
	if err != nil {
		logrus.Fatalln("Could not read configuration: ", err)
	}

	logrus.SetLevel(configuration.LogrusLevel)

	flag.Parse()
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		logrus.Infof("OIS-metrics serves in http://localhost:%s/metrics.", *metricAddr)
		logrus.Error("OIS-metrics error: ", http.ListenAndServe(":"+*metricAddr, nil))
	}()

	queueProcessor := queue.NewQueueProcessor(configuration)
	queue.UserAgentHeader = fmt.Sprintf("%s/%s %s (%s/%s)", OISVersion, OISCommitVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH)

	go func() {
		err = queueProcessor.StartProcessing()
		if err != nil {
			logrus.Fatalln(err)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-signals:
		logrus.Infof("OIS will be stopped gracefully.")
		err := queueProcessor.StopProcessing()
		if err != nil {
			logrus.Fatalln(err)
		}
	}

	os.Exit(0)
}
