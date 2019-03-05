package queue

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/oec/conf"
	"github.com/opsgenie/oec/git"
	"github.com/opsgenie/oec/runbook"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"time"
)

type QueueMessage interface {
	Message() *sqs.Message
	Process() (*runbook.ActionResultPayload, error)
}

type OECQueueMessage struct {
	message      *sqs.Message
	repositories git.Repositories
	actionSpecs  *conf.ActionSpecifications
}

func NewOECMessage(message *sqs.Message, repositories git.Repositories, actionSpecs *conf.ActionSpecifications) QueueMessage {

	return &OECQueueMessage{
		message:      message,
		repositories: repositories,
		actionSpecs:  actionSpecs,
	}
}

func (qm *OECQueueMessage) Message() *sqs.Message {
	return qm.message
}

func (qm *OECQueueMessage) Process() (*runbook.ActionResultPayload, error) {
	queuePayload := QueuePayload{}
	err := json.Unmarshal([]byte(*qm.message.Body), &queuePayload)
	if err != nil {
		return nil, err
	}

	action := queuePayload.Action
	if action == "" {
		return nil, errors.New("SQS message does not contain action property.")
	}

	mappedAction, ok := qm.actionSpecs.ActionMappings[conf.ActionName(action)]
	if !ok {
		return nil, errors.Errorf("There is no mapped action found for action[%s].", action)
	}

	result := &runbook.ActionResultPayload{
		AlertId: queuePayload.Alert.AlertId,
		Action:  queuePayload.Action,
	}

	start := time.Now()
	err = qm.execute(&mappedAction)
	took := time.Since(start)

	switch err := err.(type) {
	case *runbook.ExecError:
		result.FailureMessage = fmt.Sprintf("Err: %s, Stderr: %s", err.Error(), err.Stderr)
		logrus.Debugf("Action[%s] execution of message[%s] failed: %s Stderr: %s", action, *qm.message.MessageId, err.Error(), err.Stderr)

	case nil:
		result.IsSuccessful = true
		logrus.Debugf("Action[%s] execution of message[%s] has been completed and it took %f seconds.", action, *qm.message.MessageId, took.Seconds())

	default:
		return nil, err
	}

	return result, nil
}

func (qm *OECQueueMessage) execute(mappedAction *conf.MappedAction) error {
	args := append([]string{"-payload", *qm.message.Body}, qm.actionSpecs.GlobalFlags.Args()...)
	args = append(args, mappedAction.Flags.Args()...)
	args = append(args, qm.actionSpecs.GlobalArgs...)
	args = append(args, mappedAction.Args...)
	env := append(qm.actionSpecs.GlobalEnv, mappedAction.Env...)

	var outFile, errFile io.Writer

	if mappedAction.Stdout != "" {
		outFile = &lumberjack.Logger{
			Filename:  mappedAction.Stdout,
			MaxSize:   3, // MB
			MaxAge:    1, // Days
			LocalTime: true,
		}
	}
	if mappedAction.Stderr != "" {
		errFile = &lumberjack.Logger{
			Filename:  mappedAction.Stderr,
			MaxSize:   3, // MB
			MaxAge:    1, // Days
			LocalTime: true,
		}
	}

	sourceType := mappedAction.SourceType

	switch sourceType {
	case conf.GitSourceType:
		if qm.repositories == nil {
			return errors.New("Repositories should be provided.")
		}

		repository, err := qm.repositories.Get(mappedAction.GitOptions.Url)
		if err != nil {
			return err
		}

		repository.RLock()
		defer repository.RUnlock()
		fallthrough
	case conf.LocalSourceType:
		return runbook.ExecuteFunc(mappedAction.Filepath, args, env, outFile, errFile)
	default:
		return errors.Errorf("Unknown runbook sourceType[%s].", sourceType)
	}
}
