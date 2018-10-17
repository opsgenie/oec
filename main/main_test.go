package main

import (
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/queue"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestMainQueueError(t *testing.T) {
	queueProcessor := queue.NewQueueProcessor()
	queueProcessor.(*queue.MaridQueueProcessor).StartMethod = mockQueueProcessorStartMethod
	err := queueProcessor.Start()
	expected := errors.New("Queue processor cannot be started!").Error()
	assert.Equal(t, expected, err.Error())
}

func TestConfFileError(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "cem")
	err := conf.ReadConfFile()
	expected := errors.New("Unknown configuration source [cem].").Error()
	assert.Equal(t, expected, err.Error())
}

func mockQueueProcessorStartMethod(qp *queue.MaridQueueProcessor) error {
	return errors.New("Queue processor cannot be started!")
}
