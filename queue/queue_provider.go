package queue

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"strings"
	"sync"
)

const integrationId = "integrationId"

type SQS interface {
	ChangeMessageVisibility(input *sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error)
	DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error)
	ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
}

/************************************************************************************************/

type QueueProvider interface {
	ChangeMessageVisibility(message *sqs.Message, visibilityTimeout int64) error
	DeleteMessage(message *sqs.Message) error
	ReceiveMessage(numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error)

	RefreshClient(assumeRoleResult AssumeRoleResult) error
	OISMetadata() OISMetadata
	IsTokenExpired() bool
}

type OISQueueProvider struct {
	oisMetadata    OISMetadata
	client         SQS
	isTokenExpired bool

	refreshClientMutex *sync.RWMutex
	expirationMutex    *sync.RWMutex
}

func NewQueueProvider(oisMetadata OISMetadata) (QueueProvider, error) {
	provider := &OISQueueProvider{
		oisMetadata:        oisMetadata,
		refreshClientMutex: &sync.RWMutex{},
		expirationMutex:    &sync.RWMutex{},
	}

	err := provider.RefreshClient(oisMetadata.AssumeRoleResult)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func (qp *OISQueueProvider) OISMetadata() OISMetadata {
	qp.refreshClientMutex.RLock()
	defer qp.refreshClientMutex.RUnlock()
	return qp.oisMetadata
}

func (qp *OISQueueProvider) IsTokenExpired() bool {
	qp.expirationMutex.RLock()
	defer qp.expirationMutex.RUnlock()
	return qp.isTokenExpired
}

func (qp *OISQueueProvider) ChangeMessageVisibility(message *sqs.Message, visibilityTimeout int64) error {

	queueUrl := qp.oisMetadata.QueueUrl()

	request := &sqs.ChangeMessageVisibilityInput{
		ReceiptHandle:     message.ReceiptHandle,
		QueueUrl:          &queueUrl,
		VisibilityTimeout: &visibilityTimeout,
	}

	qp.refreshClientMutex.RLock()
	_, err := qp.client.ChangeMessageVisibility(request)
	qp.checkExpiration(err)
	qp.refreshClientMutex.RUnlock()

	if err != nil {
		return err
	}
	return nil
}

func (qp *OISQueueProvider) DeleteMessage(message *sqs.Message) error {

	queueUrl := qp.oisMetadata.QueueUrl()

	request := &sqs.DeleteMessageInput{
		QueueUrl:      &queueUrl,
		ReceiptHandle: message.ReceiptHandle,
	}

	qp.refreshClientMutex.RLock()
	_, err := qp.client.DeleteMessage(request)
	qp.checkExpiration(err)
	qp.refreshClientMutex.RUnlock()

	if err != nil {
		return err
	}
	return nil
}

func (qp *OISQueueProvider) ReceiveMessage(maxNumOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {

	queueUrl := qp.oisMetadata.QueueUrl()

	request := &sqs.ReceiveMessageInput{
		MessageAttributeNames: []*string{
			aws.String(integrationId),
		},
		QueueUrl:            &queueUrl,
		MaxNumberOfMessages: aws.Int64(maxNumOfMessage),
		VisibilityTimeout:   aws.Int64(visibilityTimeout),
		WaitTimeSeconds:     aws.Int64(0),
	}

	qp.refreshClientMutex.RLock()
	result, err := qp.client.ReceiveMessage(request)
	qp.checkExpiration(err)
	qp.refreshClientMutex.RUnlock()

	if err != nil {
		return nil, err
	}
	return result.Messages, nil
}

func (qp *OISQueueProvider) RefreshClient(assumeRoleResult AssumeRoleResult) error {

	config := qp.newConfig(assumeRoleResult)
	sess, err := session.NewSession(config)
	if err != nil {
		return err
	}

	qp.refreshClientMutex.Lock()
	qp.client = sqs.New(sess)
	qp.oisMetadata.AssumeRoleResult = assumeRoleResult
	qp.refreshClientMutex.Unlock()

	qp.expirationMutex.Lock()
	qp.isTokenExpired = false
	qp.expirationMutex.Unlock()

	return nil
}

func (qp *OISQueueProvider) newConfig(assumeRoleResult AssumeRoleResult) *aws.Config {

	assumeRoleResultCredentials := assumeRoleResult.Credentials
	creds := credentials.NewStaticCredentials(
		assumeRoleResultCredentials.AccessKeyId,
		assumeRoleResultCredentials.SecretAccessKey,
		assumeRoleResultCredentials.SessionToken,
	)

	awsConfig := aws.NewConfig().
		WithRegion(qp.oisMetadata.Region()).
		WithCredentials(creds)

	return awsConfig
}

func (qp *OISQueueProvider) checkExpiration(err error) {
	if err, ok := err.(awserr.Error); ok {
		if strings.Contains(err.Code(), "ExpiredToken") {
			qp.expirationMutex.Lock()
			qp.isTokenExpired = true
			qp.expirationMutex.Unlock()
		}
	}
}
