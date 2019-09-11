package queue

type OECToken struct {
	OwnerId         string        `json:"ownerId,omitempty"`
	OECMetadataList []OECMetadata `json:"queueProperties,omitempty"`
}

type OECMetadata struct {
	AssumeRoleResult   AssumeRoleResult   `json:"assumeRoleResult,omitempty"`
	QueueConfiguration QueueConfiguration `json:"queueConfiguration,omitempty"`
}

type AssumeRoleResult struct {
	Credentials Credentials `json:"credentials,omitempty"`
	AssumeRole  AssumeRole  `json:"assumeRole,omitempty"`
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
	Region                        string `json:"region,omitempty"`
	QueueUrl                      string `json:"queueUrl,omitempty"`
}

func (m OECMetadata) ExpireTimeMillis() int64 {
	return m.AssumeRoleResult.Credentials.ExpireTimeMillis
}

func (m OECMetadata) Region() string {
	return m.QueueConfiguration.Region
}

func (m OECMetadata) QueueUrl() string {
	return m.QueueConfiguration.QueueUrl
}
