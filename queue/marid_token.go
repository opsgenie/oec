package queue

type MaridToken struct {
	MaridMetaDataList []MaridMetadata `json:"data,omitempty"`
}

type MaridMetadata struct {
	AssumeRoleResult   AssumeRoleResult   `json:"assumeRoleResult,omitempty"`
	QueueConfiguration QueueConfiguration `json:"queueConfigurationDto,omitempty"`
}

type AssumeRoleResult struct {
	Credentials		Credentials	`json:"credentials,omitempty"`
	AssumeRole		AssumeRole	`json:"assumeRole,omitempty"`
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
	SuccessRefreshPeriodInSeconds int64  `json:"credentialSuccessRefreshPeriod,omitempty"`
	ErrorRefreshPeriodInSeconds   int64  `json:"credentialErrorRefreshPeriod,omitempty"`
	Region                        string `json:"sqsEndpoint,omitempty"`
	QueueUrl                      string `json:"queueUrl,omitempty"`
}

func (mmt *MaridMetadata) getExpireTimeMillis() int64 {
	return mmt.AssumeRoleResult.Credentials.ExpireTimeMillis
}

func (mmt *MaridMetadata) getRegion() string {
	return mmt.QueueConfiguration.Region
}

func (mmt *MaridMetadata) getQueueUrl() string {
	return mmt.QueueConfiguration.QueueUrl
}
