package main

import (
	"bytes"
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/queue"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestQueueProcessorStartError(t *testing.T) {

	buffer := &bytes.Buffer{}
	logrus.SetOutput(buffer)

	readConfFileFunc = func() (configuration *conf.Configuration, e error) {
		return &conf.Configuration{LogLevel: logrus.DebugLevel}, nil
	}

	newQueueProcessorFunc = func(conf *conf.Configuration) queue.QueueProcessor {
		return &MockQueueProcessor{startErr: errors.New("Queue processor start error")}
	}

	main()

	assert.Equal(t, logrus.DebugLevel, logrus.GetLevel())
	assert.Contains(t, buffer.String(), "Queue processor start error")
}

type MockQueueProcessor struct {
	queue.MaridQueueProcessor
	startErr error
	stopErr error
}

func (qp *MockQueueProcessor) StartProcessing() error {
	return qp.startErr
}

func (qp *MockQueueProcessor) StopProcessing() error {
	return qp.stopErr
}