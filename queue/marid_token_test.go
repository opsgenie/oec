package queue

var mockAssumeRoleResult1 = AssumeRoleResult{
	Credentials: Credentials{
		AccessKeyId:      "accessKeyId1",
		SecretAccessKey:  "secretAccessKey1",
		SessionToken:     "sessionToken1",
		ExpireTimeMillis: 123456789123,
	},
}

var mockAssumeRoleResult2 = AssumeRoleResult{
	Credentials: Credentials{
		AccessKeyId:      "accessKeyId2",
		SecretAccessKey:  "secretAccessKey2",
		SessionToken:     "sessionToken2",
		ExpireTimeMillis: 123456789123,
	},
}

var mockQueueUrl1 = "https://sqs.us-west-2.amazonaws.com/255452344566/ois-test-1-2"
var mockQueueUrl2 = "https://sqs.us-east-2.amazonaws.com/255452344566/ois-test-1-2"

var mockQueueConf1 = QueueConfiguration{
	SuccessRefreshPeriodInSeconds: 60,
	ErrorRefreshPeriodInSeconds:   60,
	Region:                        "us-west-2",
	QueueUrl:                      mockQueueUrl1,
}

var mockQueueConf2 = QueueConfiguration{
	SuccessRefreshPeriodInSeconds: 60,
	ErrorRefreshPeriodInSeconds:   60,
	Region:                        "us-east-2",
	QueueUrl:                      mockQueueUrl2,
}

var mockOISMetadata1 = OISMetadata{
	AssumeRoleResult:   mockAssumeRoleResult1,
	QueueConfiguration: mockQueueConf1,
}

var mockOISMetadata2 = OISMetadata{
	AssumeRoleResult:   mockAssumeRoleResult2,
	QueueConfiguration: mockQueueConf2,
}

var mockOISMetadataWithEmptyAssumeRoleResult1 = OISMetadata{
	AssumeRoleResult:   AssumeRoleResult{},
	QueueConfiguration: mockQueueConf1,
}

var mockOISMetadataWithEmptyAssumeRoleResult2 = OISMetadata{
	AssumeRoleResult:   AssumeRoleResult{},
	QueueConfiguration: mockQueueConf2,
}

var mockToken = OISToken{
	"12345",
	[]OISMetadata{
		mockOISMetadata1,
		mockOISMetadata2,
	},
}

var mockTokenWithEmptyAssumeRoleResult = OISToken{
	"54321",
	[]OISMetadata{
		mockOISMetadataWithEmptyAssumeRoleResult1,
		mockOISMetadataWithEmptyAssumeRoleResult2,
	},
}

var mockEmptyToken = OISToken{}
