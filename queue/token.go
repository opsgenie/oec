package queue

type token struct {
	OwnerId             string       `json:"ownerId,omitempty"`
	QueuePropertiesList []Properties `json:"queueProperties,omitempty"`
}

type Properties struct {
	AssumeRoleResult AssumeRoleResult `json:"assumeRoleResult,omitempty"`
	Configuration    Configuration    `json:"queueConfiguration,omitempty"`
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

type Configuration struct {
	SuccessRefreshPeriodInSeconds int64  `json:"credentialSuccessRefreshPeriod,omitempty"`
	ErrorRefreshPeriodInSeconds   int64  `json:"credentialErrorRefreshPeriod,omitempty"`
	Region                        string `json:"region,omitempty"`
	Url                           string `json:"queueUrl,omitempty"`
}

func (p Properties) ExpireTimeMillis() int64 {
	return p.AssumeRoleResult.Credentials.ExpireTimeMillis
}

func (p Properties) Region() string {
	return p.Configuration.Region
}

func (p Properties) Url() string {
	return p.Configuration.Url
}
