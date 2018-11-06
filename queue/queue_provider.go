package queue

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/aws/aws-sdk-go/aws/session"
	"log"
	"sync"
	"net/http"
)

const tokenUrl = "https://app.opsgenie.com/v2/integrations/maridv2/credentials"
var httpNewRequest = http.NewRequest
var newSession = session.NewSession

type QueueProvider interface {
	ChangeMessageVisibility(message *sqs.Message, visibilityTimeout int64) error
	DeleteMessage(message *sqs.Message) error
	ReceiveMessage(numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error)

	RefreshClient(assumeRoleResult *AssumeRoleResult) error
	GetMaridMetadata() *MaridMetadata
}

type MaridQueueProvider struct {

	maridMetadata *MaridMetadata

	client	*sqs.SQS
	rwMu	*sync.RWMutex

	ChangeMessageVisibilityMethod func(mqp *MaridQueueProvider, message *sqs.Message, visibilityTimeout int64) error
	DeleteMessageMethod           func(mqp *MaridQueueProvider, message *sqs.Message) error
	ReceiveMessageMethod          func(mqp *MaridQueueProvider, numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error)

	awsChangeMessageVisibilityMethod 	func(input *sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error)
	awsDeleteMessageMethod 				func(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error)
	awsReceiveMessageMethod       		func(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)

	refreshClientMethod func(mqp *MaridQueueProvider, assumeRoleResult *AssumeRoleResult) error
	newConfigMethod     func(mqp *MaridQueueProvider, assumeRoleResult *AssumeRoleResult) *aws.Config
}

func NewQueueProvider(maridMetadata *MaridMetadata) (QueueProvider, error) {
	 provider := &MaridQueueProvider {
		maridMetadata:                 maridMetadata,
		rwMu:                          &sync.RWMutex{},
		ChangeMessageVisibilityMethod: ChangeMessageVisibility,
		DeleteMessageMethod:           DeleteMessage,
		ReceiveMessageMethod:          ReceiveMessage,
		refreshClientMethod:           RefreshClient,
		newConfigMethod:               newConfig,
	}
	err := provider.RefreshClient(&maridMetadata.AssumeRoleResult)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func (mqp *MaridQueueProvider) GetMaridMetadata() *MaridMetadata {
	return mqp.maridMetadata
}

func (mqp *MaridQueueProvider) ChangeMessageVisibility(message *sqs.Message, visibilityTimeout int64) error {
	return mqp.ChangeMessageVisibilityMethod(mqp, message, visibilityTimeout)
}

func (mqp *MaridQueueProvider) DeleteMessage(message *sqs.Message) error {
	return mqp.DeleteMessageMethod(mqp, message)
}

func (mqp *MaridQueueProvider) ReceiveMessage(numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {
	return mqp.ReceiveMessageMethod(mqp, numOfMessage, visibilityTimeout)
}

func (mqp *MaridQueueProvider) RefreshClient(assumeRoleResult *AssumeRoleResult) error {
	return mqp.refreshClientMethod(mqp, assumeRoleResult)
}

func (mqp *MaridQueueProvider) newConfig(assumeRoleResult *AssumeRoleResult) *aws.Config {
	return mqp.newConfigMethod(mqp, assumeRoleResult)
}

func ChangeMessageVisibility(mqp *MaridQueueProvider, message *sqs.Message, visibilityTimeout int64) error {

	queueUrl := mqp.maridMetadata.getQueueUrl()

	request := &sqs.ChangeMessageVisibilityInput{
		ReceiptHandle:     message.ReceiptHandle,
		QueueUrl:          &queueUrl,
		VisibilityTimeout: &visibilityTimeout,
	}

	mqp.rwMu.RLock()
	resultVisibility, err := mqp.awsChangeMessageVisibilityMethod(request)
	mqp.rwMu.RUnlock()

	if err != nil {
		log.Printf("Change Message Visibility Error: %s", err)
		return err
	}

	log.Printf("Visibility Time Changed: %s", resultVisibility.String())
	return nil
}

func DeleteMessage(mqp *MaridQueueProvider, message *sqs.Message) error {

	queueUrl := mqp.maridMetadata.getQueueUrl()

	request := &sqs.DeleteMessageInput{
		QueueUrl:      &queueUrl,
		ReceiptHandle: message.ReceiptHandle,
	}

	mqp.rwMu.RLock()
	resultDelete, err := mqp.awsDeleteMessageMethod(request)
	mqp.rwMu.RUnlock()

	if err != nil {
		log.Printf("Delete message error: %s", err)
		return err
	}
	log.Printf("Message deleted: %s", resultDelete.String())
	return nil
}

func ReceiveMessage(mqp *MaridQueueProvider, maxNumOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {

	queueUrl := mqp.maridMetadata.getQueueUrl()

	request := &sqs.ReceiveMessageInput{ // todo check attributes
		AttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
		MessageAttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
		QueueUrl:            &queueUrl,
		MaxNumberOfMessages: aws.Int64(maxNumOfMessage),
		VisibilityTimeout:   aws.Int64(visibilityTimeout),
		WaitTimeSeconds:     aws.Int64(0),
	}

	mqp.rwMu.RLock()
	result, err := mqp.awsReceiveMessageMethod(request)
	mqp.rwMu.RUnlock()

	if err != nil {
		log.Printf("Receive message error: %s", err)
		return nil, err
	}

	log.Printf("Received %d messages.", len(result.Messages))

	return result.Messages, nil
}

func RefreshClient(mqp *MaridQueueProvider, assumeRoleResult *AssumeRoleResult) error {

	config := mqp.newConfig(assumeRoleResult)
	sess, err := newSession(config)
	if err != nil {
		return err
	}

	mqp.rwMu.Lock()
	mqp.client = sqs.New(sess)
	mqp.maridMetadata.AssumeRoleResult = *assumeRoleResult
	mqp.awsChangeMessageVisibilityMethod = mqp.client.ChangeMessageVisibility
	mqp.awsDeleteMessageMethod = mqp.client.DeleteMessage
	mqp.awsReceiveMessageMethod = mqp.client.ReceiveMessage
	mqp.rwMu.Unlock()

	log.Printf("Client of queue provider[%s] has refreshed.", mqp.maridMetadata.getQueueUrl())

	return nil
}


func newConfig(mqp *MaridQueueProvider, assumeRoleResult *AssumeRoleResult) *aws.Config {

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
