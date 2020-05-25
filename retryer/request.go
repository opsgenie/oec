package retryer

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
)

type Request struct {
	body io.ReadSeeker
	*http.Request
}

func NewRequest(method, url string, body io.Reader) (*Request, error) {

	rs, ok := body.(io.ReadSeeker)
	if !ok && body != nil {
		data, err := ioutil.ReadAll(body)
		if err != nil {
			return nil, err
		}
		rs = bytes.NewReader(data)
	}

	request, err := http.NewRequest(method, url, rs)
	if err != nil {
		return nil, err
	}
	return &Request{rs, request}, nil
}
