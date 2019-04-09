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

	alertId := queuePayload.Alert.AlertId
	action := queuePayload.MappedAction.Name
	if action == "" {
		action = queuePayload.Action
	}

	if action == "" {
		return nil, errors.Errorf("SQS message with alertId[%s] does not contain action property.", alertId)
	}

	mappedAction, ok := qm.actionSpecs.ActionMappings[conf.ActionName(action)]
	if !ok {
		return nil, errors.Errorf("There is no mapped action found for action[%s]. SQS message with alertId[%s] will be ignored.", action, alertId)
	}

	result := &runbook.ActionResultPayload{
		AlertId: alertId,
		Action:  action,
	}

	start := time.Now()
	err = qm.execute(&mappedAction)
	took := time.Since(start)

	switch err := err.(type) {
	case *runbook.ExecError:
		result.FailureMessage = fmt.Sprintf("Err: %s, Stderr: %s", err.Error(), err.Stderr)
		logrus.Debugf("Action[%s] execution of message[%s] with alertId[%s] failed: %s Stderr: %s", action, *qm.message.MessageId, alertId, err.Error(), err.Stderr)

	case nil:
		result.IsSuccessful = true
		logrus.Debugf("Action[%s] execution of message[%s] with alertId[%s] has been completed and it took %f seconds.", action, *qm.message.MessageId, alertId, took.Seconds())

	default:
		return nil, err
	}

	return result, nil
}

func (qm *OECQueueMessage) execute(mappedAction *conf.MappedAction) error {

	args := append(qm.actionSpecs.GlobalFlags.Args(), mappedAction.Flags.Args()...)
	args = append(args, []string{"-payload", *qm.message.Body}...)
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
