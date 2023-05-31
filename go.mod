module github.com/polarismesh/polaris

go 1.18

require (
	github.com/BurntSushi/toml v1.2.0
	github.com/boltdb/bolt v1.3.1
	github.com/emicklei/go-restful/v3 v3.9.0
	github.com/envoyproxy/go-control-plane v0.10.2-0.20220325020618-49ff273808a1
	github.com/go-openapi/spec v0.20.7
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.3.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/golang-lru v0.5.4
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mitchellh/mapstructure v1.4.3
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/nicksnyder/go-i18n/v2 v2.2.0
	github.com/pkg/errors v0.9.1
	github.com/polarismesh/go-restful-openapi/v2 v2.0.0-20220928152401-083908d10219
	github.com/prometheus/client_golang v1.12.2
	github.com/smartystreets/goconvey v1.6.4
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.8.2
	go.uber.org/atomic v1.10.0
	go.uber.org/automaxprocs v1.4.0
	go.uber.org/zap v1.23.0
	golang.org/x/crypto v0.1.0
	golang.org/x/net v0.2.0 // indirect
	golang.org/x/sync v0.1.0
	golang.org/x/text v0.4.0
	golang.org/x/time v0.1.1-0.20221020023724-80b9fac54d29
	google.golang.org/grpc v1.51.0
	google.golang.org/protobuf v1.28.1
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/ArthurHlt/go-eureka-client v1.1.0
	github.com/google/go-cmp v0.5.8 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.3 // indirect
	github.com/sirupsen/logrus v1.6.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

// Indirect dependencies group
require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cncf/xds/go v0.0.0-20211130200136-a8f946100490 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/envoyproxy/protoc-gen-validate v0.6.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20191106031601-ce3c9ade29de // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.uber.org/goleak v1.1.12 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
	google.golang.org/genproto v0.0.0-20221014213838-99cd37c6964a // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/agiledragon/gomonkey/v2 v2.10.1
	github.com/polarismesh/specification v1.3.2-alpha.2
)

require gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect

replace gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.2
