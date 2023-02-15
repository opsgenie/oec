module github.com/opsgenie/oec

go 1.12

replace (
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	golang.org/x/text => golang.org/x/text v0.3.4
)

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/aws/aws-sdk-go v1.23.20
	github.com/go-git/go-git/v5 v5.2.0
	github.com/google/uuid v1.1.1
	github.com/kardianos/service v1.0.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.1
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.4.0
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0-20170531160350-a96e63847dc3
	gopkg.in/yaml.v2 v2.4.0
)
