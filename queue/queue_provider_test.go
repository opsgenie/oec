package queue

import (
	"bytes"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"testing"
)

func mockHttpGetError(url string) (*http.Response, error) {
	return nil, errors.New("Test http error has occurred while getting token.")
}

func mockHttpGet(url string) (*http.Response, error) {
	response := &http.Response{}
	response.Body = ioutil.NopCloser(
		bytes.NewBufferString(`
	{"Credentials": {"AccessKeyId": "testAccessKeyId", "SecretAccessKey": "secretkjndkf", "SessionToken": "kjhfds", "ExpireTimeMillis": 5},
	"AssumedRole": {"Id": "id123", "Arn": "arnarnarn"},
	"OGQueueConfiguration": {"SuccessRefreshPeriod": 20, "ErrorRefreshPeriod": 5, "SqsEndpoint": "us-east-2", "QueueUrl": "queueUrl"} }`))
	return response, nil
}

func mockSuccessChange(mqp *MaridQueueProvider, message *sqs.Message, visibilityTimeout int64) error {
	return nil
}

func mockSuccessDelete(mqp *MaridQueueProvider, message *sqs.Message) error {
	return nil
}

func TestGetToken(t *testing.T) {

	httpGetMethod = mockHttpGet

	mqp := MaridQueueProvider{
		newTokenMethod: newToken,
	}

	ogPayload, err := mqp.newToken(httpGetMethod)

	assert.Nil(t, err)
	assert.Equal(t, ogPayload.Data.AssumeRoleResult.Credentials.AccessKeyId, "testAccessKeyId")
}

func TestGetTokenError(t *testing.T) {

	httpGetMethod = mockHttpGetError

	mqp := MaridQueueProvider{
		newTokenMethod: newToken,
	}

	_, err := mqp.newToken(httpGetMethod)

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
