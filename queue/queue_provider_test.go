package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"testing"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pkg/errors"
	"strconv"
	"time"
	"math/rand"
	"sync"
)

var testMqp = NewQueueProviderForTest(mockMaridMetadata1).(*MaridQueueProvider)
var defaultMqp = NewQueueProviderForTest(mockMaridMetadata1).(*MaridQueueProvider)

func NewQueueProviderForTest(maridMetadata MaridMetadata) QueueProvider {
	return &MaridQueueProvider {
		maridMetadata:                 &maridMetadata,
		rwMu:                          &sync.RWMutex{},
		ChangeMessageVisibilityMethod: ChangeMessageVisibility,
		DeleteMessageMethod:           DeleteMessage,
		ReceiveMessageMethod:          ReceiveMessage,
		refreshClientMethod:           RefreshClient,
		newConfigMethod:               newConfig,
	}
}

var mockAssumeRoleResult = mockAssumeRoleResult1
var mockCreds = credentials.NewStaticCredentials(
	mockAssumeRoleResult.Credentials.AccessKeyId,
	mockAssumeRoleResult.Credentials.SecretAccessKey,
	mockAssumeRoleResult.Credentials.SessionToken)

func mockChangeMessageVisibilitySuccessOfProvider(mqp *MaridQueueProvider, message *sqs.Message, visibilityTimeout int64) error {
	return nil
}

func mockDeleteMessageSuccessOfProvider(mqp *MaridQueueProvider, message *sqs.Message) error {
	return nil
}

func mockReceiveMessageSuccessOfProvider(mqp *MaridQueueProvider, numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {
	messages := make([]*sqs.Message, 0)
	for i := int64(0); i < numOfMessage ; i++ {
		sqsMessage := &sqs.Message{}
		sqsMessage.SetMessageId(strconv.Itoa(int(i+1)))
		messages = append(messages, sqsMessage)
	}
	return messages, nil
}

func mockReceiveZeroMessage(mqp *MaridQueueProvider, numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {
	return []*sqs.Message{}, nil
}

func mockReceiveMessageError(mqp *MaridQueueProvider, numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {
	return nil, errors.New("Test receive message error")
}

func mockPoll(p *PollerImpl) (shouldWait bool) {
	for j:=0; j< 10 ; j++ {
		for i := 0; i < 10 ; i++ {
			if !p.isWorkerPoolRunning() {
				return
			}
			sqsMessage := &sqs.Message{}
			sqsMessage.SetMessageId(strconv.Itoa(j*10+(i+1)))
			message := NewMaridMessage(sqsMessage)
			job := NewSqsJob(message, p.queueProvider, 1)
			p.submit(job)
		}
		time.Sleep(time.Millisecond * 10)
	}

	return rand.Intn(2) == 0
}

func TestChangeMessageVisibility(t *testing.T) {

	defer func() {
		testMqp.awsChangeMessageVisibilityMethod = nil
	}()

	testMqp.awsChangeMessageVisibilityMethod = func(input *sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error) {
		result := &sqs.ChangeMessageVisibilityOutput{}
		return result, nil
	}

	err := testMqp.ChangeMessageVisibility(&sqs.Message{ReceiptHandle: new(string)}, 15)

	assert.Nil(t, err)
}

func TestTestChangeMessageVisibilityWithError(t *testing.T) {

	defer func() {
		testMqp.awsChangeMessageVisibilityMethod = nil
	}()

	testMqp.awsChangeMessageVisibilityMethod = func(input *sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error) {
		return nil, errors.New("Test change message visibility error")
	}

	err := testMqp.ChangeMessageVisibility(&sqs.Message{ReceiptHandle: new(string)}, 15)

	assert.NotNil(t, err)
	assert.Equal(t, "Test change message visibility error", err.Error())
}

func TestDeleteMessage(t *testing.T) {

	defer func() {
		testMqp.awsDeleteMessageMethod = nil
	}()

	testMqp.awsDeleteMessageMethod = func(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
		result := &sqs.DeleteMessageOutput{}
		return result, nil
	}

	err := testMqp.DeleteMessage(&sqs.Message{ReceiptHandle: new(string)})

	assert.Nil(t, err)
}

func TestDeleteMessageWithError(t *testing.T) {

	defer func() {
		testMqp.awsDeleteMessageMethod = nil
	}()

	testMqp.awsDeleteMessageMethod = func(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
		return nil, errors.New("Test delete message error")
	}

	err := testMqp.DeleteMessage(&sqs.Message{ReceiptHandle: new(string)})

	assert.NotNil(t, err)
	assert.Equal(t, "Test delete message error", err.Error())
}

func TestReceiveMessage(t *testing.T) {

	defer func() {
		testMqp.awsReceiveMessageMethod = nil
	}()

	testMqp.awsReceiveMessageMethod = func(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
		result := &sqs.ReceiveMessageOutput{
			Messages: []*sqs.Message{},
		}
		return result, nil
	}

	messages, err := testMqp.ReceiveMessage(5, 15)

	assert.Nil(t, err)
	assert.Equal(t, 0, len(messages))
}

func TestReceiveMessageWithError(t *testing.T) {

	defer func() {
		testMqp.awsReceiveMessageMethod = nil
	}()

	testMqp.awsReceiveMessageMethod = func(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
		return nil, errors.New("Test receive message error")
	}

	_, err := testMqp.ReceiveMessage(5, 15)

	assert.NotNil(t, err)
	assert.Equal(t, "Test receive message error", err.Error())
}

func TestRefreshClient(t *testing.T) {

	err := testMqp.RefreshClient(&mockAssumeRoleResult)

	assert.Nil(t, err)

	expected := aws.NewConfig().
		WithRegion(testMqp.GetMaridMetadata().getRegion()).
		WithCredentials(mockCreds)

	assert.Equal(t, expected.Credentials, testMqp.client.Config.Credentials)
	assert.Equal(t, expected.Region, testMqp.client.Config.Region)
}

func TestRefreshClientWithNilConfig(t *testing.T) {

	defer func() {
		newSession = session.NewSession
	}()

	newSession = func(cfgs ...*aws.Config) (*session.Session, error) {
		return nil, errors.New("Test new session error")
	}

	err := testMqp.RefreshClient(&mockAssumeRoleResult)

	assert.NotNil(t, err)
}

func TestNewConfig(t *testing.T) {

	assert.Equal(t, mockMaridMetadata1, *testMqp.GetMaridMetadata())

	expected := aws.NewConfig().
		WithRegion(testMqp.GetMaridMetadata().getRegion()).
		WithCredentials(mockCreds)

	awsConfig := testMqp.newConfig(&mockAssumeRoleResult)
	assert.Equal(t, expected, awsConfig)

	expected.WithRegion("wrongRegion")
	assert.NotEqual(t, expected, awsConfig)
}
