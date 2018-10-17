package queue

import (
	"strconv"
)

type OGPayload struct {
	Data Data `json:"data,omitempty"`
}

type Data struct {
	AssumeRoleResult AssumeRoleResult `json:"assumeRoleResult,omitempty"`
}

type AssumeRoleResult struct {
	Credentials          Credentials          `json:"credentials,omitempty"`
	AssumeRole           AssumeRole           `json:"assumeRole,omitempty"`
	OGQueueConfiguration OGQueueConfiguration `json:"queueConfigurationDto,omitempty"`
}

type Credentials struct {
	AccessKeyId      string `json:"accessKeyId,omitempty"`
	SecretAccessKey  string `json:"secretAccessKey,omitempty"`
	SessionToken     string `json:"sessionToken,omitempty"`
	ExpireTimeMillis int64  `json:"expireTimeMillis,omitempty"`
}

type AssumeRole struct {
	Id  string `json:"id,omitempty"`
	Arn string `json:"arn,omitempty"`
}

type OGQueueConfiguration struct { // todo change name
	SuccessRefreshPeriod int64  `json:"credentialSuccessRefreshPeriod,omitempty"`
	ErrorRefreshPeriod   int64  `json:"credentialErrorRefreshPeriod,omitempty"`
	SqsEndpoint          string `json:"sqsEndpoint,omitempty"`
	QueueUrl             string `json:"queueUrl,omitempty"`
}

func (og *OGPayload) toString() string {
	return "Credentials: " + "{" + og.Data.AssumeRoleResult.Credentials.AccessKeyId + "," + og.Data.AssumeRoleResult.Credentials.SecretAccessKey + "," + og.Data.AssumeRoleResult.Credentials.SessionToken + "," + strconv.FormatInt(og.Data.AssumeRoleResult.Credentials.ExpireTimeMillis, 10) + "}\n" +
		"AssumedRole: " + "{" + og.Data.AssumeRoleResult.AssumeRole.Id + "," + og.Data.AssumeRoleResult.AssumeRole.Arn + "}"
}

func (og *OGPayload) getEndpoint() string {
	queueUrl := og.Data.AssumeRoleResult.OGQueueConfiguration.SqsEndpoint
	return queueUrl
}

func (og *OGPayload) getQueueUrl() string {
	queueUrl := og.Data.AssumeRoleResult.OGQueueConfiguration.QueueUrl
	return queueUrl
}

func (og *OGPayload) getSuccessRefreshPeriod() int64 {
	successRefreshPeriod := og.Data.AssumeRoleResult.OGQueueConfiguration.SuccessRefreshPeriod
	return successRefreshPeriod
}

func (og *OGPayload) getErrorRefreshPeriod() int64 {
	errorRefreshPeriod := og.Data.AssumeRoleResult.OGQueueConfiguration.SuccessRefreshPeriod
	return errorRefreshPeriod
}
