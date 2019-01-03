package queue

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"
	"sync"
)

const integrationId = "integrationId"

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
	IsTokenExpired() bool
}

type MaridQueueProvider struct {

	maridMetadata 		MaridMetadata
	client             	SQS
	isTokenExpired     	bool
	refreshClientMutex 	*sync.RWMutex
}

func NewQueueProvider(maridMetadata MaridMetadata) (QueueProvider, error) {
	provider := &MaridQueueProvider {
		maridMetadata:      maridMetadata,
		refreshClientMutex: &sync.RWMutex{},
	}

	err := provider.RefreshClient(maridMetadata.AssumeRoleResult)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func (mqp *MaridQueueProvider) MaridMetadata() MaridMetadata {
	mqp.refreshClientMutex.RLock()
	defer mqp.refreshClientMutex.RUnlock()
	return mqp.maridMetadata
}

func (mqp *MaridQueueProvider) IsTokenExpired() bool {
	mqp.refreshClientMutex.RLock()
	defer mqp.refreshClientMutex.RUnlock()
	return mqp.isTokenExpired
}

func (mqp *MaridQueueProvider) ChangeMessageVisibility(message *sqs.Message, visibilityTimeout int64) error {

	queueUrl := mqp.maridMetadata.QueueUrl()

	request := &sqs.ChangeMessageVisibilityInput{
		ReceiptHandle:     message.ReceiptHandle,
		QueueUrl:          &queueUrl,
		VisibilityTimeout: &visibilityTimeout,
	}

	mqp.refreshClientMutex.RLock()
	_, err := mqp.client.ChangeMessageVisibility(request)
	mqp.refreshClientMutex.RUnlock()

	mqp.checkExpiration(err)

	if err != nil {
		return err
	}
	return nil
}

func (mqp *MaridQueueProvider) DeleteMessage(message *sqs.Message) error {

	queueUrl := mqp.maridMetadata.QueueUrl()

	request := &sqs.DeleteMessageInput{
		QueueUrl:      &queueUrl,
		ReceiptHandle: message.ReceiptHandle,
	}

	mqp.refreshClientMutex.RLock()
	_, err := mqp.client.DeleteMessage(request)
	mqp.refreshClientMutex.RUnlock()

	mqp.checkExpiration(err)

	if err != nil {
		return err
	}
	return nil
}

func (mqp *MaridQueueProvider) ReceiveMessage(maxNumOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {

	queueUrl := mqp.maridMetadata.QueueUrl()

	request := &sqs.ReceiveMessageInput{
		MessageAttributeNames: []*string{
			aws.String(integrationId),
		},
		QueueUrl:            &queueUrl,
		MaxNumberOfMessages: aws.Int64(maxNumOfMessage),
		VisibilityTimeout:   aws.Int64(visibilityTimeout),
		WaitTimeSeconds:     aws.Int64(0),
	}

	mqp.refreshClientMutex.RLock()
	result, err := mqp.client.ReceiveMessage(request)
	mqp.refreshClientMutex.RUnlock()

	mqp.checkExpiration(err)

	if err != nil {
		return nil, err
	}
	return result.Messages, nil
}

func (mqp *MaridQueueProvider) RefreshClient(assumeRoleResult AssumeRoleResult) error {

	config := mqp.newConfig(assumeRoleResult)
	sess, err := session.NewSession(config)
	if err != nil {
		return err
	}

	mqp.refreshClientMutex.Lock()
	mqp.client = sqs.New(sess)
	mqp.maridMetadata.AssumeRoleResult = assumeRoleResult
	mqp.isTokenExpired = false
	mqp.refreshClientMutex.Unlock()

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
		WithRegion(mqp.maridMetadata.Region()).
		WithCredentials(creds)

	return awsConfig
}

func (mqp *MaridQueueProvider) checkExpiration(err error) {
	if err, ok := err.(awserr.Error); ok {
		if err.Code() == sts.ErrCodeExpiredTokenException {
			mqp.refreshClientMutex.Lock()
			mqp.isTokenExpired = true
			mqp.refreshClientMutex.Unlock()
		}
	}
}
