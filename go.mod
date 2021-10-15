module github.com/polarismesh/polaris-server

go 1.12

require (
	github.com/boltdb/bolt v1.3.1
	github.com/emicklei/go-restful v2.9.6+incompatible
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.4.3
	github.com/gomodule/redigo v1.8.5
	github.com/google/uuid v1.2.0
	github.com/hashicorp/golang-lru v0.5.3
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/mitchellh/mapstructure v1.1.2
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/pkg/errors v0.8.1
	github.com/smartystreets/goconvey v0.0.0-20190710185942-9d28bd7c0945
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.5.1
	go.uber.org/atomic v1.5.1 // indirect
	go.uber.org/multierr v1.4.0 // indirect
	go.uber.org/zap v1.14.0
	golang.org/x/lint v0.0.0-20191125180803-fdd1cda4f05f // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0
	google.golang.org/grpc v1.36.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v2 v2.2.8
	github.com/envoyproxy/go-control-plane v0.9.9

)

replace gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.2
