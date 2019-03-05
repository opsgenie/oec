package main

import (
	"flag"
	"fmt"
	"github.com/opsgenie/oec/conf"
	"github.com/opsgenie/oec/queue"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"
)

var metricAddr = flag.String("oec-metrics", "7070", "The address to listen on for HTTP requests.")
var defaultLogFilepath = filepath.Join("/var", "log", "opsgenie", "oec"+strconv.Itoa(os.Getpid())+".log")

var OECVersion string
var OECCommitVersion string

func main() {

	logrus.SetFormatter(
		&logrus.TextFormatter{
			ForceColors:     true,
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		},
	)

	err := os.Chmod(filepath.Join("/var", "log"), 0644)
	if err != nil {
		logrus.Warn(err)
	}

	logger := &lumberjack.Logger{
		Filename:  defaultLogFilepath,
		MaxSize:   3,  // MB
		MaxAge:    10, // Days
		LocalTime: true,
	}

	logrus.SetOutput(io.MultiWriter(os.Stdout, logger))

	logrus.Infof("OEC version is %s", OECVersion)
	logrus.Infof("OEC commit version is %s", OECCommitVersion)

	go checkLogFile(logger, time.Second*10)

	configuration, err := conf.ReadConfFile()
	if err != nil {
		logrus.Fatalf("Could not read configuration: %s", err)
	}

	logrus.SetLevel(configuration.LogrusLevel)

	flag.Parse()
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		logrus.Infof("OEC-metrics serves in http://localhost:%s/metrics.", *metricAddr)
		logrus.Error("OEC-metrics error: ", http.ListenAndServe(":"+*metricAddr, nil))
	}()

	queueProcessor := queue.NewQueueProcessor(configuration)
	queue.UserAgentHeader = fmt.Sprintf("%s/%s %s (%s/%s)", OECVersion, OECCommitVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH)

	go func() {
		if configuration.AppName != "" {
			logrus.Infof("%s is starting.", configuration.AppName)
		}
		err = queueProcessor.StartProcessing()
		if err != nil {
			logrus.Fatalln(err)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-signals:
		logrus.Infof("OEC will be stopped gracefully.")
		err := queueProcessor.StopProcessing()
		if err != nil {
			logrus.Fatalln(err)
		}
	}

	os.Exit(0)
}

func checkLogFile(logger *lumberjack.Logger, interval time.Duration) {
	for {
		select {
		case <-time.After(interval):
			if _, err := os.Stat(defaultLogFilepath); os.IsNotExist(err) {
				logrus.Warnf("Failed to open OEC log file: %v. New file will be created.", err)
				if err = logger.Rotate(); err != nil {
					logrus.Warn(err)
				} else {
					logrus.Warnf("New OEC log file is created, previous one might be removed accidentally.")
				}
				break
			}
		}
	}
}
