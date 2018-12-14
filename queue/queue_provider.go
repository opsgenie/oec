package queue

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/sirupsen/logrus"
	"sync"
)

//const tokenUrl = "https://api.opsgenie.com/v2/integrations/maridv2/credentials"
const tokenUrl = "https://1c2365a0.ngrok.io/v2/integrations/maridv2/credentials"

var newSession = session.NewSession

type SQS interface {
	DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error)
	ChangeMessageVisibility(input *sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error)
	ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
}

/************************************************************************************************/

type QueueProvider interface {
	ChangeMessageVisibility(message *sqs.Message, visibilityTimeout int64) error
	DeleteMessage(message *sqs.Message) error
	ReceiveMessage(numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error)

	RefreshClient(assumeRoleResult AssumeRoleResult) error
	MaridMetadata() MaridMetadata
	IntegrationId() string
}

type MaridQueueProvider struct {

	maridMetadata MaridMetadata
	integrationId string

	client	SQS
	rwMu	*sync.RWMutex
}

func NewQueueProvider(maridMetadata MaridMetadata, integrationId string) (QueueProvider, error) {
	 provider := &MaridQueueProvider {
		maridMetadata:	maridMetadata,
		integrationId:	integrationId,
		rwMu:			&sync.RWMutex{},
	}
	err := provider.RefreshClient(maridMetadata.AssumeRoleResult)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func (mqp *MaridQueueProvider) IntegrationId() string {
	return mqp.integrationId
}

func (mqp *MaridQueueProvider) MaridMetadata() MaridMetadata {
	return mqp.maridMetadata
}

func (mqp *MaridQueueProvider) ChangeMessageVisibility(message *sqs.Message, visibilityTimeout int64) error {

	queueUrl := mqp.maridMetadata.getQueueUrl()

	request := &sqs.ChangeMessageVisibilityInput{
		ReceiptHandle:     message.ReceiptHandle,
		QueueUrl:          &queueUrl,
		VisibilityTimeout: &visibilityTimeout,
	}

	mqp.rwMu.RLock()
	_, err := mqp.client.ChangeMessageVisibility(request)
	mqp.rwMu.RUnlock()

	if err != nil {
		logrus.Errorf("Change Message Visibility Error: %s", err)
		return err
	}

	logrus.Debugf("Visibility time of Message[%s] is changed.", *message.MessageId)
	return nil
}

func (mqp *MaridQueueProvider) DeleteMessage(message *sqs.Message) error {

	queueUrl := mqp.maridMetadata.getQueueUrl()

	request := &sqs.DeleteMessageInput{
		QueueUrl:      &queueUrl,
		ReceiptHandle: message.ReceiptHandle,
	}

	mqp.rwMu.RLock()
	_, err := mqp.client.DeleteMessage(request)
	mqp.rwMu.RUnlock()

	if err != nil {
		logrus.Errorf("Delete message error: %s", err)
		return err
	}
	logrus.Debugf("Message[%s] is deleted.", *message.MessageId)
	return nil
}

func (mqp *MaridQueueProvider) ReceiveMessage(maxNumOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {

	queueUrl := mqp.maridMetadata.getQueueUrl()

	request := &sqs.ReceiveMessageInput{
		MessageAttributeNames: []*string{
			aws.String("integrationId"),
		},
		QueueUrl:            &queueUrl,
		MaxNumberOfMessages: aws.Int64(maxNumOfMessage),
		VisibilityTimeout:   aws.Int64(visibilityTimeout),
		WaitTimeSeconds:     aws.Int64(0),
	}

	mqp.rwMu.RLock()
	result, err := mqp.client.ReceiveMessage(request)
	mqp.rwMu.RUnlock()

	if err != nil {
		logrus.Errorf("Receive message error: %s", err)
		return nil, err
	}

	logrus.Debugf("Received %d messages.", len(result.Messages))

	return result.Messages, nil
}

func (mqp *MaridQueueProvider) RefreshClient(assumeRoleResult AssumeRoleResult) error {

	config := mqp.newConfig(assumeRoleResult)
	sess, err := newSession(config)
	if err != nil {
		return err
	}

	mqp.rwMu.Lock()
	mqp.client = sqs.New(sess)
	mqp.maridMetadata.AssumeRoleResult = assumeRoleResult
	mqp.rwMu.Unlock()

	logrus.Debugf("Client of queue provider[%s] has refreshed.", mqp.maridMetadata.getQueueUrl())

	return nil
}


func (mqp *MaridQueueProvider) newConfig(assumeRoleResult AssumeRoleResult) *aws.Config {

	assumeRoleResultCredentials := assumeRoleResult.Credentials
	creds := credentials.NewStaticCredentials(
		assumeRoleResultCredentials.AccessKeyId,
		assumeRoleResultCredentials.SecretAccessKey,
		assumeRoleResultCredentials.SessionToken,
	)

	awsConfig := aws.NewConfig().
		WithRegion(mqp.maridMetadata.getRegion()).
		WithCredentials(creds)

	return awsConfig
}
