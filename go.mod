module github.com/opsgenie/oec

go 1.12

replace (
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	golang.org/x/text => golang.org/x/text v0.3.4
)

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/alcortesm/tgz v0.0.0-20161220082320-9c5fe88206d7 // indirect
	github.com/aws/aws-sdk-go v1.23.20
	github.com/flynn/go-shlex v0.0.0-20150515145356-3f9db97f8568 // indirect
	github.com/go-git/go-git/v5 v5.11.0
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.1.1
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/jessevdk/go-flags v1.4.0 // indirect
	github.com/kardianos/service v1.0.0
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.1
	github.com/sirupsen/logrus v1.9.0
	github.com/stretchr/testify v1.8.4
	golang.org/x/net v0.23.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0-20170531160350-a96e63847dc3
	gopkg.in/yaml.v2 v2.4.0
)
