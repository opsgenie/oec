package queue

import (
	"net/http"
	"encoding/json"
	"bytes"
	"io/ioutil"
	"github.com/pkg/errors"
)

func mockHttpGetError(retryer *Retryer, request *http.Request) (*http.Response, error) {
	return nil, errors.New("Test http error has occurred while getting token.")
}

func mockHttpGet(retryer *Retryer, request *http.Request) (*http.Response, error) {

	token, _ := json.Marshal(mockToken)
	buff := bytes.NewBuffer(token)

	response := &http.Response{}
	response.Body = ioutil.NopCloser(buff)

	return response, nil
}

func mockHttpGetInvalidJson(retryer *Retryer, request *http.Request) (*http.Response, error) {

	response := &http.Response{}
	response.Body = ioutil.NopCloser( bytes.NewBufferString(`{"Invalid json": }`))

	return response, nil
}
