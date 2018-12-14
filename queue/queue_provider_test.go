package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"strconv"
	"sync"
	"time"
	"math/rand"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/pkg/errors"
)

func newQueueProviderTest() *MaridQueueProvider {
	return &MaridQueueProvider {
		maridMetadata:	mockMaridMetadata1,
		integrationId:	"mockIntegrationId",
		rwMu:			&sync.RWMutex{},
		client:			NewMockSqsClient(nil),
	}
}

var mockAssumeRoleResult = mockAssumeRoleResult2
var mockCreds = credentials.NewStaticCredentials(
	mockAssumeRoleResult.Credentials.AccessKeyId,
	mockAssumeRoleResult.Credentials.SecretAccessKey,
	mockAssumeRoleResult.Credentials.SessionToken)

var mockReceiptHandle = "mockReceiptHandle"

func mockReceiveMessageSuccessOfProvider(numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {
	messages := make([]*sqs.Message, 0)
	for i := int64(0); i < numOfMessage ; i++ {
		sqsMessage := &sqs.Message{}
		sqsMessage.SetMessageId(strconv.Itoa(int(i+1)))
		messages = append(messages, sqsMessage)
	}
	return messages, nil
}

func mockPoll(p *MaridPoller) (shouldWait bool) {
	for j:=0; j< 10 ; j++ {
		for i := 0; i < 10 ; i++ {
			if !p.workerPool.IsRunning() {
				return
			}
			sqsMessage := &sqs.Message{}
			sqsMessage.SetMessageId(strconv.Itoa(j*10+(i+1)))
			message := NewMaridMessage(sqsMessage, mockActionMappings, &mockApiKey)
			job := NewSqsJob(message, p.queueProvider, 1)
			p.workerPool.Submit(job)
		}
		time.Sleep(time.Millisecond * 10)
	}

	return rand.Intn(2) == 0
}

func TestChangeMessageVisibility(t *testing.T) {

	provider := newQueueProviderTest()

	var capturedInput *sqs.ChangeMessageVisibilityInput
	provider.client.(*mockSqsClient).ChangeMessageVisibilityFunc = func(input *sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error) {
		capturedInput = input
		return nil, nil
	}

	err := provider.ChangeMessageVisibility(&sqs.Message{ReceiptHandle: &mockReceiptHandle, MessageId: new(string)}, 0)

	assert.Nil(t, err)
	assert.Equal(t, mockReceiptHandle, *capturedInput.ReceiptHandle)
	assert.Equal(t, int64(0), *capturedInput.VisibilityTimeout)
	assert.Equal(t, mockQueueUrl1, *capturedInput.QueueUrl)
}

func TestChangeMessageVisibilityWithError(t *testing.T) {

	provider := newQueueProviderTest()

	provider.client.(*mockSqsClient).ChangeMessageVisibilityFunc = func(input *sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error) {
		return nil, errors.New("Test change message visibility error")
	}

	err := provider.ChangeMessageVisibility(&sqs.Message{ReceiptHandle: &mockReceiptHandle, MessageId: new(string)}, 0)

	assert.NotNil(t, err)
	assert.Equal(t, "Test change message visibility error", err.Error())
}

func TestDeleteMessage(t *testing.T) {

	provider := newQueueProviderTest()

	var capturedInput *sqs.DeleteMessageInput
	provider.client.(*mockSqsClient).DeleteMessageFunc = func(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
		capturedInput = input
		return nil, nil
	}

	err := provider.DeleteMessage(&sqs.Message{ReceiptHandle: &mockReceiptHandle, MessageId: new(string)})

	assert.Nil(t, err)
	assert.Equal(t, mockReceiptHandle, *capturedInput.ReceiptHandle)
	assert.Equal(t, mockQueueUrl1, *capturedInput.QueueUrl)
}

func TestDeleteMessageWithError(t *testing.T) {

	provider := newQueueProviderTest()

	provider.client.(*mockSqsClient).DeleteMessageFunc = func(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
		return nil, errors.New("Test delete message error")
	}

	err := provider.DeleteMessage(&sqs.Message{ReceiptHandle: &mockReceiptHandle, MessageId: new(string)})

	assert.NotNil(t, err)
	assert.Equal(t, "Test delete message error", err.Error())
}

func TestReceiveMessage(t *testing.T) {

	provider := newQueueProviderTest()

	var capturedInput *sqs.ReceiveMessageInput
	provider.client.(*mockSqsClient).ReceiveMessageFunc = func(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
		capturedInput = input
		return &sqs.ReceiveMessageOutput{Messages: []*sqs.Message{{},{}}}, nil
	}

	messages, err := provider.ReceiveMessage(10, 30)

	assert.Nil(t, err)
	assert.Equal(t, 2, len(messages))
	assert.Equal(t, int64(30), *capturedInput.VisibilityTimeout)
	assert.Equal(t, mockQueueUrl1, *capturedInput.QueueUrl)
	assert.Equal(t, int64(0), *capturedInput.WaitTimeSeconds)	// because of short polling
	assert.Equal(t, int64(10), *capturedInput.MaxNumberOfMessages)
	assert.Equal(t, 1, len(capturedInput.MessageAttributeNames))
	assert.Equal(t, "integrationId", *capturedInput.MessageAttributeNames[0])
}

func TestReceiveMessageWithError(t *testing.T) {

	provider := newQueueProviderTest()

	provider.client.(*mockSqsClient).ReceiveMessageFunc = func(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
		return nil, errors.New("Test receive message visibility error")
	}

	_, err := provider.ReceiveMessage(10,30)

	assert.NotNil(t, err)
	assert.Equal(t, "Test receive message visibility error", err.Error())
}

func TestRefreshClient(t *testing.T) {

	provider := newQueueProviderTest()

	err := provider.RefreshClient(mockAssumeRoleResult2)

	assert.Nil(t, err)

	expectedConfig := aws.NewConfig().
		WithRegion(provider.MaridMetadata().getRegion()).
		WithCredentials(mockCreds)

	assert.Equal(t, expectedConfig.Credentials, provider.client.(*sqs.SQS).Config.Credentials)
	assert.Equal(t, expectedConfig.Region, provider.client.(*sqs.SQS).Config.Region)
	assert.Equal(t, mockAssumeRoleResult2, provider.maridMetadata.AssumeRoleResult)
}

// Mock SqsClient
type mockSqsClient struct {
	DeleteMessageFunc func(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error)
	ChangeMessageVisibilityFunc func(input *sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error)
	ReceiveMessageFunc func(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
}

func (c *mockSqsClient) DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	if c.DeleteMessageFunc != nil {
		return c.DeleteMessageFunc(input)
	}
	return nil, nil
}

func (c *mockSqsClient) ChangeMessageVisibility(input *sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error) {
	if c.ChangeMessageVisibilityFunc != nil {
		return c.ChangeMessageVisibilityFunc(input)
	}
	return nil, nil
}

func (c *mockSqsClient) ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	if c.ReceiveMessageFunc != nil {
		return c.ReceiveMessageFunc(input)
	}
	return &sqs.ReceiveMessageOutput{Messages: []*sqs.Message{}}, nil // empty slice of message
}

func NewMockSqsClient(p client.ConfigProvider, cfgs ...*aws.Config) SQS {
	return new(mockSqsClient)
}

// Mock QueueProvider
type MockQueueProvider struct {

	ChangeMessageVisibilityFunc func(message *sqs.Message, visibilityTimeout int64) error
	DeleteMessageFunc func(message *sqs.Message) error
	ReceiveMessageFunc func(numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error)
	MaridMetadataFunc func() MaridMetadata
	IntegrationIdFunc func() string
	RefreshClientFunc func(assumeRoleResult AssumeRoleResult) error
}

func NewMockQueueProvider() QueueProvider {
	return &MockQueueProvider{
	}
}

func (mqp *MockQueueProvider) IntegrationId() string {
	if mqp.IntegrationIdFunc != nil {
		return mqp.IntegrationIdFunc()
	}
	return "mockIntegrationId"
}

func (mqp *MockQueueProvider) ChangeMessageVisibility(message *sqs.Message, visibilityTimeout int64) error {
	if mqp.ChangeMessageVisibilityFunc != nil {
		return mqp.ChangeMessageVisibilityFunc(message, visibilityTimeout)
	}
	return nil
}

func (mqp *MockQueueProvider) DeleteMessage(message *sqs.Message) error {
	if mqp.DeleteMessageFunc != nil {
		return mqp.DeleteMessageFunc(message)
	}
	return nil
}

func (mqp *MockQueueProvider) ReceiveMessage(numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {
	if mqp.ReceiveMessageFunc != nil {
		return mqp.ReceiveMessageFunc(numOfMessage, visibilityTimeout)
	}
	return []*sqs.Message{}, nil
}

func (mqp *MockQueueProvider) MaridMetadata() MaridMetadata {
	if mqp.MaridMetadataFunc != nil {
		return mqp.MaridMetadataFunc()
	}
	return mockMaridMetadata1
}

func (mqp *MockQueueProvider) RefreshClient(assumeRoleResult AssumeRoleResult) error {
	if mqp.RefreshClientFunc != nil {
		return mqp.RefreshClientFunc(assumeRoleResult)
	}
	return nil
}

var mockSuccessReceiveFunc = func(numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {
	messages := make([]*sqs.Message,0)
	for i := int64(0); i < numOfMessage; i++ {
		id := strconv.FormatInt(i, 10)
		integrationId := "mockIntegrationId"
		messageAttr := map[string]*sqs.MessageAttributeValue{"integrationId": {StringValue: &integrationId} }
		messages = append(messages, &sqs.Message{MessageId: &id, MessageAttributes: messageAttr })
	}

	return messages, nil
}
