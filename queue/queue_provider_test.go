package queue

import (
	"bytes"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"testing"
	"github.com/opsgenie/marid2/conf"
	"encoding/json"
)

var testMqp = NewQueueProvider().(*MaridQueueProvider)
var defaultMqp = NewQueueProvider().(*MaridQueueProvider)


func mockHttpGetError(retryer *Retryer, request *http.Request) (*http.Response, error) {
	return nil, errors.New("Test http error has occurred while getting token.")
}

func mockHttpGet(retryer *Retryer, request *http.Request) (*http.Response, error) {

	payload, _ := json.Marshal(OGPayload{
		Data: Data{
			AssumeRoleResult: AssumeRoleResult{
				Credentials: Credentials{
					AccessKeyId: "testAccessKeyId",
					SecretAccessKey: "secret",
					SessionToken: "session",
					ExpireTimeMillis: 1234,
				},
			},
			QueueConfiguration: QueueConfiguration{
				SqsEndpoint: "us-east-2",
			},
		},
	})
	buff := bytes.NewBuffer(payload)

	response := &http.Response{}
	response.Body = ioutil.NopCloser(buff)

	return response, nil
}

func mockSuccessChange(mqp *MaridQueueProvider, message *sqs.Message, visibilityTimeout int64) error {
	return nil
}

func mockSuccessDelete(mqp *MaridQueueProvider, message *sqs.Message) error {
	return nil
}

func TestReceiveToken(t *testing.T) {

	defer func() {
		testMqp.retryer.getMethod = defaultMqp.retryer.getMethod
		conf.Configuration = make(map[string]interface{})
	}()

	testMqp.retryer.getMethod = mockHttpGet

	conf.Configuration = map[string]interface{}{
		"apiKey" : "",
	}

	ogPayload, err := testMqp.newToken()

	assert.Nil(t, err)
	assert.Equal(t, "testAccessKeyId", ogPayload.Data.AssumeRoleResult.Credentials.AccessKeyId)
}

func TestReceiveTokenError(t *testing.T) {

	defer func() {
		testMqp.retryer.getMethod = defaultMqp.retryer.getMethod
		conf.Configuration = make(map[string]interface{})
	}()

	testMqp.retryer.getMethod = mockHttpGetError

	conf.Configuration = map[string]interface{}{
		"apiKey" : "",
	}

	_, err := testMqp.newToken()

	assert.NotNil(t, err)
	assert.Equal(t, "Test http error has occurred while getting token.", err.Error())
}

/*func TestNewConfig(t *testing.T) {
	var s SqsService
	s.client,_ = s.newClient(s.newConfig(""))

}

func TestNewClient(t *testing.T) {
	client, err := newClient()
	credentials.newS
	assert.NoError(t, err)
	assert.NotNil(t, client)
}*/
