package queue

var mockToken = MaridToken {
	Data: Data {
		AssumeRoleResult: AssumeRoleResult {
			Credentials: Credentials {
				AccessKeyId: "accessKeyId",
				SecretAccessKey: "secretAccessKey",
				SessionToken: "sessionToken",
				ExpireTimeMillis: 123456789123,
			},
		},
		QueueConfiguration: QueueConfiguration {
			SqsEndpoint: "us-east-2",
			QueueUrls:	[]string{
				"https://sqs.us-west-2.amazonaws.com/255452344566/marid-test-1-2",
				"https://sqs.us-east-2.amazonaws.com/255452344566/marid-test-1-2",
			},
		},
	},
}

var mockTokenWithEmptyAssumeRoleResult = MaridToken {
	Data: Data {
		AssumeRoleResult: AssumeRoleResult {},
		QueueConfiguration: QueueConfiguration {
			SqsEndpoint: "us-east-2",
			QueueUrls:	[]string{
				"https://sqs.us-west-2.amazonaws.com/255452344566/marid-test-1-2",
				"https://sqs.us-east-2.amazonaws.com/255452344566/marid-test-1-2",
			},
		},
	},
}

var mockEmptyToken = MaridToken {}


