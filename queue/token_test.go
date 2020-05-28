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

var mockQueueUrl1 = "https://sqs.us-west-2.amazonaws.com/255452344566/oec-test-1-2"
var mockQueueUrl2 = "https://sqs.us-east-2.amazonaws.com/255452344566/oec-test-1-2"

var mockQueueConf1 = Configuration{
	SuccessRefreshPeriodInSeconds: 60,
	ErrorRefreshPeriodInSeconds:   60,
	Region:                        "us-west-2",
	Url:                           mockQueueUrl1,
}

var mockQueueConf2 = Configuration{
	SuccessRefreshPeriodInSeconds: 60,
	ErrorRefreshPeriodInSeconds:   60,
	Region:                        "us-east-2",
	Url:                           mockQueueUrl2,
}

var mockQueueProperties1 = Properties{
	AssumeRoleResult: mockAssumeRoleResult1,
	Configuration:    mockQueueConf1,
}

var mockQueueProperties2 = Properties{
	AssumeRoleResult: mockAssumeRoleResult2,
	Configuration:    mockQueueConf2,
}

var mockQueuePropertiesWithEmptyAssumeRoleResult1 = Properties{
	AssumeRoleResult: AssumeRoleResult{},
	Configuration:    mockQueueConf1,
}

var mockQueuePropertiesWithEmptyAssumeRoleResult2 = Properties{
	AssumeRoleResult: AssumeRoleResult{},
	Configuration:    mockQueueConf2,
}

var mockToken = token{
	"12345",
	[]Properties{
		mockQueueProperties1,
		mockQueueProperties2,
	},
}

var mockTokenWithEmptyAssumeRoleResult = token{
	"54321",
	[]Properties{
		mockQueuePropertiesWithEmptyAssumeRoleResult1,
		mockQueuePropertiesWithEmptyAssumeRoleResult2,
	},
}

var mockEmptyToken = token{}
