package queue


var mockAssumeRoleResult1 = AssumeRoleResult{
	Credentials: Credentials{
		AccessKeyId:      "accessKeyId1",
		SecretAccessKey:  "secretAccessKey1",
		SessionToken:     "sessionToken1",
		ExpireTimeMillis: 123456789123,
	},
}

var mockAssumeRoleResult2 = AssumeRoleResult {
	Credentials: Credentials {
		AccessKeyId:      "accessKeyId2",
		SecretAccessKey:  "secretAccessKey2",
		SessionToken:     "sessionToken2",
		ExpireTimeMillis: 123456789123,
	},
}

var mockQueueUrl1 = "https://sqs.us-west-2.amazonaws.com/255452344566/marid-test-1-2"
var mockQueueUrl2 = "https://sqs.us-east-2.amazonaws.com/255452344566/marid-test-1-2"

var mockQueueConf1 = QueueConfiguration {
	SuccessRefreshPeriodInSeconds: 60,
	ErrorRefreshPeriodInSeconds: 60,
	Region: "us-west-2",
	QueueUrl: mockQueueUrl1,
}

var mockQueueConf2 = QueueConfiguration {
	SuccessRefreshPeriodInSeconds: 60,
	ErrorRefreshPeriodInSeconds: 60,
	Region: "us-east-2",
	QueueUrl: mockQueueUrl2,
}

var mockMaridMetadata1 = MaridMetadata{
	AssumeRoleResult: mockAssumeRoleResult1,
	QueueConfiguration: mockQueueConf1,
}

var mockMaridMetadata2 = MaridMetadata{
	AssumeRoleResult: mockAssumeRoleResult2,
	QueueConfiguration: mockQueueConf2,
}

var mockMaridMetadataWithEmptyAssumeRoleResult1 = MaridMetadata{
	AssumeRoleResult: AssumeRoleResult{},
	QueueConfiguration: mockQueueConf1,
}

var mockMaridMetadataWithEmptyAssumeRoleResult2 = MaridMetadata{
	AssumeRoleResult: AssumeRoleResult{},
	QueueConfiguration: mockQueueConf2,
}

var mockToken = MaridToken {
	"12345",
	[]MaridMetadata{
		mockMaridMetadata1,
		mockMaridMetadata2,
	},
}

var mockTokenWithEmptyAssumeRoleResult = MaridToken {
	"54321",
	[]MaridMetadata{
		mockMaridMetadataWithEmptyAssumeRoleResult1,
		mockMaridMetadataWithEmptyAssumeRoleResult2,
	},
}

var mockEmptyToken = MaridToken{}


