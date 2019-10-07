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

const ownerId = "ownerId"

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
	OECMetadata() OECMetadata
	IsTokenExpired() bool
}

type OECQueueProvider struct {
	oecMetadata    OECMetadata
	client         SQS
	isTokenExpired bool

	refreshClientMu *sync.RWMutex
	expirationMu    *sync.RWMutex
}

func NewQueueProvider(oecMetadata OECMetadata) (QueueProvider, error) {
	provider := &OECQueueProvider{
		oecMetadata:     oecMetadata,
		refreshClientMu: &sync.RWMutex{},
		expirationMu:    &sync.RWMutex{},
	}

	err := provider.RefreshClient(oecMetadata.AssumeRoleResult)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func (qp *OECQueueProvider) OECMetadata() OECMetadata {
	qp.refreshClientMu.RLock()
	defer qp.refreshClientMu.RUnlock()
	return qp.oecMetadata
}

func (qp *OECQueueProvider) IsTokenExpired() bool {
	qp.expirationMu.RLock()
	defer qp.expirationMu.RUnlock()
	return qp.isTokenExpired
}

func (qp *OECQueueProvider) ChangeMessageVisibility(message *sqs.Message, visibilityTimeout int64) error {

	queueUrl := qp.oecMetadata.QueueUrl()

	request := &sqs.ChangeMessageVisibilityInput{
		ReceiptHandle:     message.ReceiptHandle,
		QueueUrl:          &queueUrl,
		VisibilityTimeout: &visibilityTimeout,
	}

	qp.refreshClientMu.RLock()
	_, err := qp.client.ChangeMessageVisibility(request)
	qp.checkExpiration(err)
	qp.refreshClientMu.RUnlock()

	if err != nil {
		return err
	}
	return nil
}

func (qp *OECQueueProvider) DeleteMessage(message *sqs.Message) error {

	queueUrl := qp.oecMetadata.QueueUrl()

	request := &sqs.DeleteMessageInput{
		QueueUrl:      &queueUrl,
		ReceiptHandle: message.ReceiptHandle,
	}

	qp.refreshClientMu.RLock()
	_, err := qp.client.DeleteMessage(request)
	qp.checkExpiration(err)
	qp.refreshClientMu.RUnlock()

	if err != nil {
		return err
	}
	return nil
}

func (qp *OECQueueProvider) ReceiveMessage(maxNumOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {

	queueUrl := qp.oecMetadata.QueueUrl()

	request := &sqs.ReceiveMessageInput{
		MessageAttributeNames: []*string{
			aws.String(ownerId),
		},
		QueueUrl:            &queueUrl,
		MaxNumberOfMessages: aws.Int64(maxNumOfMessage),
		VisibilityTimeout:   aws.Int64(visibilityTimeout),
		WaitTimeSeconds:     aws.Int64(0),
	}

	qp.refreshClientMu.RLock()
	result, err := qp.client.ReceiveMessage(request)
	qp.checkExpiration(err)
	qp.refreshClientMu.RUnlock()

	if err != nil {
		return nil, err
	}
	return result.Messages, nil
}

func (qp *OECQueueProvider) RefreshClient(assumeRoleResult AssumeRoleResult) error {

	config := qp.newConfig(assumeRoleResult)
	sess, err := session.NewSession(config)
	if err != nil {
		return err
	}

	qp.refreshClientMu.Lock()
	qp.client = sqs.New(sess)
	qp.oecMetadata.AssumeRoleResult = assumeRoleResult
	qp.refreshClientMu.Unlock()

	qp.expirationMu.Lock()
	qp.isTokenExpired = false
	qp.expirationMu.Unlock()

	return nil
}

func (qp *OECQueueProvider) newConfig(assumeRoleResult AssumeRoleResult) *aws.Config {

	assumeRoleResultCredentials := assumeRoleResult.Credentials
	creds := credentials.NewStaticCredentials(
		assumeRoleResultCredentials.AccessKeyId,
		assumeRoleResultCredentials.SecretAccessKey,
		assumeRoleResultCredentials.SessionToken,
	)

	awsConfig := aws.NewConfig().
		WithRegion(qp.oecMetadata.Region()).
		WithCredentials(creds)

	return awsConfig
}

func (qp *OECQueueProvider) checkExpiration(err error) {
	if err, ok := err.(awserr.Error); ok {
		if strings.Contains(err.Code(), "ExpiredToken") {
			qp.expirationMu.Lock()
			qp.isTokenExpired = true
			qp.expirationMu.Unlock()
		}
	}
}
