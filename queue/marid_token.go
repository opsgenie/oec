package queue

import (
	"strconv"
)

type MaridToken struct {
	Data Data `json:"data,omitempty"`
}

type Data struct {
	AssumeRoleResult   AssumeRoleResult   `json:"assumeRoleResult,omitempty"`
	QueueConfiguration QueueConfiguration `json:"queueConfigurationDto,omitempty"`
}

type AssumeRoleResult struct {
	Credentials          Credentials          `json:"credentials,omitempty"`
	AssumeRole           AssumeRole           `json:"assumeRole,omitempty"`
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

type QueueConfiguration struct {
	SuccessRefreshPeriod int64  `json:"credentialSuccessRefreshPeriod,omitempty"`
	ErrorRefreshPeriod   int64  `json:"credentialErrorRefreshPeriod,omitempty"`
	SqsEndpoint          string `json:"sqsEndpoint,omitempty"`
	QueueUrls            []string `json:"queueUrl,omitempty"`
}

func (og *MaridToken) toString() string {
	return "Credentials: " + "{" + og.Data.AssumeRoleResult.Credentials.AccessKeyId + "," + og.Data.AssumeRoleResult.Credentials.SecretAccessKey + "," + og.Data.AssumeRoleResult.Credentials.SessionToken + "," + strconv.FormatInt(og.Data.AssumeRoleResult.Credentials.ExpireTimeMillis, 10) + "}\n" +
		"AssumedRole: " + "{" + og.Data.AssumeRoleResult.AssumeRole.Id + "," + og.Data.AssumeRoleResult.AssumeRole.Arn + "}"
}

func (og *MaridToken) getEndpoint() string {
	queueUrl := og.Data.QueueConfiguration.SqsEndpoint
	return queueUrl
}

func (og *MaridToken) getQueueUrl() []string {
	queueUrls := og.Data.QueueConfiguration.QueueUrls
	return queueUrls
}
