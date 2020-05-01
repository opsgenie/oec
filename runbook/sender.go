package runbook

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/opsgenie/oec/retryer"
	"github.com/pkg/errors"
)

const resultPath = "/v2/integrations/oec/actionExecutionResult"

var SendResultToOpsGenieFunc = SendResultToOpsGenie

var client = &retryer.Retryer{}

func SendResultToOpsGenie(resultPayload *ActionResultPayload, apiKey, baseUrl string) error {

	body, err := json.Marshal(resultPayload)
	if err != nil {
		return errors.Errorf("Cannot marshall payload: %s", err)
	}

	resultUrl := baseUrl + resultPath

	request, err := retryer.NewRequest("POST", resultUrl, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Add("Authorization", "GenieKey "+apiKey)
	request.Header.Add("Content-Type", "application/json; charset=UTF-8")

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {

		errorMessage := "Unexpected response status: " + strconv.Itoa(response.StatusCode)

		body, err := ioutil.ReadAll(response.Body)
		if err == nil {
			return errors.Errorf("%s, error message: %s", errorMessage, string(body))
		} else {
			return errors.Errorf("%s, also could not read response body: %s", errorMessage, err)
		}
	}

	return nil
}
