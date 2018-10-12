package queue

import (
	"strconv"
)

type AssumeRoleResult struct {
	Credentials Credentials `json:"Credentials,omitempty"`
	AssumedRole AssumedRole `json:"AssumedRole,omitempty"`
	OGCredentials OGCredentials `json:"OGCredentials,omitempty"`
}

type Credentials struct {
	AccessKeyId 	string	`json:"AccessKeyId,omitempty"`
	SecretAccessKey string	`json:"SecretAccessKey,omitempty"`
	SessionToken    string	`json:"SessionToken,omitempty"`
	ExpireTimeMillis int64	`json:"ExpireTimeMillis,omitempty"`
}

type AssumedRole struct {
	Id  string	`json:"Id,omitempty"`
	Arn string	`json:"Arn,omitempty"`
}

type OGCredentials struct { // todo change name
	SuccessRefreshPeriod int64 `json:"SuccessRefreshPeriod,omitempty"`
	ErrorRefreshPeriod	 int64 `json:"ErrorRefreshPeriod,omitempty"`
	SqsEndpoint	 string 	   `json:"SqsEndpoint,omitempty"`
	QueueUrl	 string		   `json:"QueueUrl,omitempty"`
}

func (arr *AssumeRoleResult) toString() string {
	return "Credentials: " + "{" + arr.Credentials.AccessKeyId + "," + arr.Credentials.SecretAccessKey + "," + arr.Credentials.SessionToken + "," + strconv.FormatInt(arr.Credentials.ExpireTimeMillis,10) + "}\n" +
		"AssumedRole: " + "{" + arr.AssumedRole.Id + "," + arr.AssumedRole.Arn + "}"
}

func (arr *AssumeRoleResult) getEndpoint() string {
	queueUrl := arr.OGCredentials.SqsEndpoint
	return queueUrl
}

func (arr *AssumeRoleResult) getQueueUrl() string {
	queueUrl := arr.OGCredentials.QueueUrl
	return queueUrl
}

func (arr *AssumeRoleResult) getSuccessRefreshPeriod() int64 {
	successRefreshPeriod := arr.OGCredentials.SuccessRefreshPeriod
	return successRefreshPeriod
}

func (arr *AssumeRoleResult) getErrorRefreshPeriod() int64 {
	errorRefreshPeriod := arr.OGCredentials.SuccessRefreshPeriod
	return errorRefreshPeriod
}