package queue

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/aws/credentials"

	"github.com/aws/aws-sdk-go/aws/session"
	"encoding/json"
	"time"
	"sync/atomic"
	"github.com/pkg/errors"
	"log"
	"fmt"
	"sync"
	"github.com/google/uuid"
)

const tokenUrl = "https://app.opsgenie.com/v2/integration/maridV2/credentials"

type QueueProvider interface {
	ChangeMessageVisibility(message *sqs.Message, visibilityTimeout int64) (error)
	DeleteMessage(message *sqs.Message) (error)
	ReceiveMessage(numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error)

	StartRefreshing() error
	StopRefreshing() error
}

type MaridQueueProvider struct {
	queueName        	string
	client           	*sqs.SQS
	assumeRoleResult 	*AssumeRoleResult
	//creds            	*credentials.Credentials

	quit         chan struct{}
	isRefreshing atomic.Value
	retryer      *Retryer
	rwMu         *sync.RWMutex
	startStopMu  *sync.Mutex

	ChangeMessageVisibilityMethod 	func(mqp *MaridQueueProvider, message *sqs.Message, visibilityTimeout int64) (error)
	DeleteMessageMethod 			func(mqp *MaridQueueProvider, message *sqs.Message) (error)
	ReceiveMessageMethod 			func(mqp *MaridQueueProvider, numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error)

	StartRefreshingMethod 			func(mqp *MaridQueueProvider) error
	StopRefreshingMethod 			func(mqp *MaridQueueProvider) error

	refreshClientMethod  			func(mqp *MaridQueueProvider, fn getMethod) (error)
	runMethod 						func(mqp *MaridQueueProvider)
	newTokenMethod 					func(mqp *MaridQueueProvider, fn getMethod) (*AssumeRoleResult, error)
	newConfigMethod 				func(mqp *MaridQueueProvider, assumeRoleResult *AssumeRoleResult) (*aws.Config)

	httpGetMethod getMethod
}

func NewQueueProvider() QueueProvider {
	qp := &MaridQueueProvider{
		quit:                          make(chan struct{}),
		retryer:                       NewRetryer(),
		queueName:                     uuid.New().String(),
		rwMu:                          &sync.RWMutex{},
		startStopMu:                   &sync.Mutex{},
		ChangeMessageVisibilityMethod: ChangeMessageVisibility,
		DeleteMessageMethod:           DeleteMessage,
		ReceiveMessageMethod:          ReceiveMessage,
		StartRefreshingMethod:         StartRefreshing,
		StopRefreshingMethod:          StopRefreshing,
		refreshClientMethod:           refreshClient,
		runMethod:                     runQueueProvider,
		newTokenMethod:                newToken,
		newConfigMethod:               newConfig,
	}
	qp.isRefreshing.Store(false)
	return qp
}

func (mqp *MaridQueueProvider) getRegion() string {
	defer mqp.rwMu.RUnlock()
	mqp.rwMu.RLock()
	return mqp.assumeRoleResult.getEndpoint()
}
func (mqp *MaridQueueProvider) getQueueUrl() string {
	defer mqp.rwMu.RUnlock()
	mqp.rwMu.RLock()
	return mqp.assumeRoleResult.getQueueUrl()
}
func (mqp *MaridQueueProvider) getSuccessPeriod() time.Duration {
	defer mqp.rwMu.RUnlock()
	mqp.rwMu.RLock()
	return time.Duration(mqp.assumeRoleResult.getSuccessRefreshPeriod()) * time.Second
}
func (mqp *MaridQueueProvider) getErrorPeriod() time.Duration {
	defer mqp.rwMu.RUnlock()
	mqp.rwMu.RLock()
	return time.Duration(mqp.assumeRoleResult.getErrorRefreshPeriod()) * time.Second
}

func (mqp *MaridQueueProvider) StartRefreshing() error {
	return mqp.StartRefreshingMethod(mqp)
}

func (mqp *MaridQueueProvider) StopRefreshing() error {
	return mqp.StopRefreshingMethod(mqp)
}

func (mqp *MaridQueueProvider) ChangeMessageVisibility(message *sqs.Message, visibilityTimeout int64) (error) {
	return mqp.ChangeMessageVisibilityMethod(mqp, message, visibilityTimeout)
}

func (mqp *MaridQueueProvider) DeleteMessage(message *sqs.Message) (error) {
	return mqp.DeleteMessageMethod(mqp, message)
}

func (mqp *MaridQueueProvider) ReceiveMessage(numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {
	return mqp.ReceiveMessageMethod(mqp, numOfMessage, visibilityTimeout)
}

func (mqp *MaridQueueProvider) refreshClient(fn getMethod) error {
	return mqp.refreshClientMethod(mqp, fn)
}

func (mqp *MaridQueueProvider) run() {
	mqp.runMethod(mqp)
}

func (mqp *MaridQueueProvider) newToken(fn getMethod) (*AssumeRoleResult, error){
	return mqp.newTokenMethod(mqp, fn)
}

func (mqp *MaridQueueProvider) newConfig(assumeRoleResult *AssumeRoleResult) (*aws.Config) {
	return mqp.newConfigMethod(mqp, assumeRoleResult)
}

func StartRefreshing(mqp *MaridQueueProvider) error {
	defer mqp.startStopMu.Unlock()
	mqp.startStopMu.Lock()

	if mqp.isRefreshing.Load().(bool) {
		return errors.New("Queue provider is already running.")
	}

	log.Printf("Queue provider[%s] is starting to refresh client.", mqp.queueName)
	if err := mqp.refreshClient(mqp.retryer.getMethod); err != nil {
		log.Printf("Queue provider[%s] could not get initial token and will terminate.", mqp.queueName)
		return err
	}

	mqp.isRefreshing.Store(true)	// todo ?

	go mqp.run()

	return nil
}

func StopRefreshing(mqp *MaridQueueProvider) error {
	defer mqp.startStopMu.Unlock()
	mqp.startStopMu.Lock()

	if !mqp.isRefreshing.Load().(bool) {
		return errors.New("Queue provider is already running.")
	}
	mqp.isRefreshing.Store(false)

	log.Printf("Queue provider[%s] is stopping to refresh client.", mqp.queueName)
	close(mqp.quit)

	return nil
}

func ChangeMessageVisibility(mqp *MaridQueueProvider, message *sqs.Message, visibilityTimeout int64) (error) {
	queueUrl := mqp.getQueueUrl()
	request := &sqs.ChangeMessageVisibilityInput {
		ReceiptHandle:     message.ReceiptHandle,
		QueueUrl:          &queueUrl,
		VisibilityTimeout: &visibilityTimeout,
	}

	mqp.rwMu.RLock()
	resultVisibility, err := mqp.client.ChangeMessageVisibility(request)
	mqp.rwMu.RUnlock()

	if err != nil {
		log.Println("Change Message Visibility Error", err)
		return err
	}

	log.Printf("Visibility Time Changed: %s", resultVisibility.String())
	return nil
}

func DeleteMessage(mqp *MaridQueueProvider, message *sqs.Message) (error) {
	queueUrl := mqp.getQueueUrl()
	request := &sqs.DeleteMessageInput{
		QueueUrl:      &queueUrl,
		ReceiptHandle: message.ReceiptHandle,
	}

	mqp.rwMu.RLock()
	resultDelete, err := mqp.client.DeleteMessage(request)
	mqp.rwMu.RUnlock()

	if err != nil {
		log.Printf("Delete message error: %s", err)
		return err
	}
	log.Printf("Message deleted: %s", resultDelete.String())
	return nil
}

func ReceiveMessage(mqp *MaridQueueProvider, maxNumOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {
	queueUrl := mqp.getQueueUrl()
	request := &sqs.ReceiveMessageInput{	// todo check attributes
		AttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
		MessageAttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
		QueueUrl:            &queueUrl,
		MaxNumberOfMessages: aws.Int64(maxNumOfMessage),
		VisibilityTimeout:   aws.Int64(visibilityTimeout),
		WaitTimeSeconds:     aws.Int64(0),	// todo check short polling
	}

	mqp.rwMu.RLock()
	result, err := mqp.client.ReceiveMessage(request)
	mqp.rwMu.RUnlock()

	if err != nil {
		fmt.Printf("Receive message error: %s", err)
		return nil, err
	}

	log.Printf("Received %d messages.", len(result.Messages))

	return result.Messages, nil
}

func refreshClient(mqp *MaridQueueProvider, fn getMethod) error {
	assumeRoleResult, err := mqp.newToken(fn)
	if err != nil {
		return err
	}
	config := mqp.newConfig(assumeRoleResult)
	sess, err := session.NewSession(config)
	if err != nil {
		return err
	}

	defer mqp.rwMu.Unlock()
	mqp.rwMu.Lock()
	mqp.assumeRoleResult = assumeRoleResult
	mqp.client = sqs.New(sess)

	return nil
}

func runQueueProvider(mqp *MaridQueueProvider) {

	log.Printf("Queue provider[%s] has started to refresh client by %s periods.", mqp.queueName, mqp.getSuccessPeriod().String())

	ticker := time.NewTicker(mqp.getSuccessPeriod())

	for {
		select {
		case <- mqp.quit:
			ticker.Stop()
			log.Printf("Queue provider[%s] has stopped to refresh client.", mqp.queueName)
			return
		case <-ticker.C:
			ticker.Stop()
			err := mqp.refreshClient(mqp.retryer.getMethod)
			if err != nil {
				log.Printf("Refresh cycle of queue provider[%s] has failed: %s", mqp.queueName, err.Error())
				ticker = time.NewTicker(mqp.getErrorPeriod())
			} else {
				log.Printf("Client of queue provider[%s] has refreshed", mqp.queueName)
				ticker = time.NewTicker(mqp.getSuccessPeriod())
			}
		}
	}
}

func newToken(mqp *MaridQueueProvider, fn getMethod) (*AssumeRoleResult, error) { // todo change name of the function
	if fn == nil {
		errors.New("please provide http get method")
	}

	response, err := fn(tokenUrl)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	assumeRoleResult := &AssumeRoleResult{}
	err = json.NewDecoder(response.Body).Decode(&assumeRoleResult)
	if err != nil {
		return nil, err
	}

	return assumeRoleResult, nil
}

func newConfig(mqp *MaridQueueProvider, assumeRoleResult *AssumeRoleResult) (*aws.Config) {

	ARRCredentials := assumeRoleResult.Credentials
	creds := credentials.NewStaticCredentials(ARRCredentials.AccessKeyId, ARRCredentials.SecretAccessKey, ARRCredentials.SessionToken)

	region := assumeRoleResult.getEndpoint()
	awsConfig := aws.NewConfig().WithRegion(region).WithCredentials(creds)

	return awsConfig
}
