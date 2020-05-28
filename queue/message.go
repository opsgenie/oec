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
	"io"
	"time"
)

type MessageHandler interface {
	Handle(message sqs.Message) (*runbook.ActionResultPayload, error)
}

type messageHandler struct {
	repositories  git.Repositories
	actionSpecs   conf.ActionSpecifications
	actionLoggers map[string]io.Writer
}

func NewMessageHandler(repositories git.Repositories, actionSpecs conf.ActionSpecifications, actionLoggers map[string]io.Writer) MessageHandler {
	return &messageHandler{
		repositories:  repositories,
		actionSpecs:   actionSpecs,
		actionLoggers: actionLoggers,
	}
}

func (mh *messageHandler) Handle(message sqs.Message) (*runbook.ActionResultPayload, error) {
	queuePayload := payload{}
	err := json.Unmarshal([]byte(*message.Body), &queuePayload)
	if err != nil {
		return nil, err
	}

	entityId := queuePayload.Entity.Id
	entityType := queuePayload.Entity.Type
	action := queuePayload.MappedAction.Name
	if action == "" {
		action = queuePayload.Action
	}

	if action == "" {
		return nil, errors.Errorf("SQS message with entityId[%s] does not contain action property.", entityId)
	}

	mappedAction, ok := mh.actionSpecs.ActionMappings[conf.ActionName(action)]
	if !ok {
		return nil, errors.Errorf("There is no mapped action found for action[%s]. SQS message with entityId[%s] will be ignored.", action, entityId)
	}

	result := &runbook.ActionResultPayload{
		EntityId:   entityId,
		EntityType: entityType,
		Action:     action,
	}

	start := time.Now()
	err = mh.execute(&mappedAction, *message.Body)
	took := time.Since(start)

	switch err := err.(type) {
	case *runbook.ExecError:
		result.FailureMessage = fmt.Sprintf("Err: %s, Stderr: %s", err.Error(), err.Stderr)
		logrus.Debugf("Action[%s] execution of message[%s] with entityId[%s] failed: %s Stderr: %s", action, *message.MessageId, entityId, err.Error(), err.Stderr)

	case nil:
		result.IsSuccessful = true
		logrus.Debugf("Action[%s] execution of message[%s] with entityId[%s] has been completed and it took %f seconds.", action, *message.MessageId, entityId, took.Seconds())

	default:
		return nil, err
	}

	return result, nil
}

func (mh *messageHandler) execute(mappedAction *conf.MappedAction, messageBody string) error {

	sourceType := mappedAction.SourceType
	switch sourceType {
	case conf.GitSourceType:
		if mh.repositories == nil {
			return errors.New("Repositories should be provided.")
		}

		repository, err := mh.repositories.Get(mappedAction.GitOptions.Url)
		if err != nil {
			return err
		}

		repository.RLock()
		defer repository.RUnlock()
		fallthrough

	case conf.LocalSourceType:
		args := append(mh.actionSpecs.GlobalFlags.Args(), mappedAction.Flags.Args()...)
		args = append(args, []string{"-payload", messageBody}...)
		args = append(args, mh.actionSpecs.GlobalArgs...)
		args = append(args, mappedAction.Args...)
		env := append(mh.actionSpecs.GlobalEnv, mappedAction.Env...)

		stdout := mh.actionLoggers[mappedAction.Stdout]
		stderr := mh.actionLoggers[mappedAction.Stderr]

		return runbook.ExecuteFunc(mappedAction.Filepath, args, env, stdout, stderr)
	default:
		return errors.Errorf("Unknown action sourceType[%s].", sourceType)
	}
}
