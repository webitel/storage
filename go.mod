module github.com/webitel/storage

go 1.24.1

toolchain go1.24.3

require (
	cloud.google.com/go/speech v1.26.0
	cloud.google.com/go/storage v1.54.0
	cloud.google.com/go/texttospeech v1.11.0
	github.com/BoRuDar/configuration/v4 v4.2.2
	github.com/aws/aws-sdk-go v1.55.5
	github.com/go-gorp/gorp v2.2.0+incompatible
	github.com/golang/protobuf v1.5.4
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.3
	github.com/h2non/filetype v1.1.3
	github.com/jmoiron/sqlx v1.4.0
	github.com/juju/ratelimit v1.0.2
	github.com/lib/pq v1.10.9
	github.com/makiuchi-d/gozxing v0.1.1
	github.com/pborman/uuid v1.2.1
	github.com/pkg/errors v0.9.1
	github.com/rabbitmq/amqp091-go v1.10.0
	github.com/robfig/cron v1.2.0
	github.com/webitel/engine/pkg/discovery v0.0.0-20250508122206-b7fb2a4de82e
	github.com/webitel/engine/pkg/presign v0.0.0-20250508102602-4109efdb5e30
	github.com/webitel/engine/pkg/wbt v0.0.0-20250508102602-4109efdb5e30
	github.com/webitel/engine/pkg/werror v0.0.0-20250508121332-6ae1563235d8
	github.com/webitel/logger v0.0.0-20250521215819-71dea53fff02
	github.com/webitel/webitel-go-kit v0.0.21-0.20250527124757-a379b4376a46
	github.com/webitel/wlog v0.0.0-20250325101442-de4f125c1ec7
	go.opentelemetry.io/otel v1.35.0
	go.opentelemetry.io/otel/sdk v1.35.0
	golang.org/x/crypto v0.38.0
	golang.org/x/image v0.12.0
	golang.org/x/sync v0.14.0
	google.golang.org/api v0.233.0
	google.golang.org/genproto v0.0.0-20250303144028-a0af3efb3deb
	google.golang.org/genproto/googleapis/api v0.0.0-20250505200425-f936aa4a68b2
	google.golang.org/grpc v1.72.1
	google.golang.org/protobuf v1.36.6
)

require (
	cel.dev/expr v0.20.0 // indirect
	cloud.google.com/go v0.121.0 // indirect
	cloud.google.com/go/auth v0.16.1 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.6.0 // indirect
	cloud.google.com/go/iam v1.5.2 // indirect
	cloud.google.com/go/longrunning v0.6.7 // indirect
	cloud.google.com/go/monitoring v1.24.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.27.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.51.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.51.0 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cncf/xds/go v0.0.0-20250121191232-2f005788dc42 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.32.4 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/form v3.1.4+incompatible // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.14.1 // indirect
	github.com/hashicorp/consul/api v1.32.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.5.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mbobakov/grpc-consul-resolver v1.5.3 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/poy/onpar v1.1.2 // indirect
	github.com/spiffe/go-spiffe/v2 v2.5.0 // indirect
	github.com/zeebo/errs v1.4.0 // indirect
	github.com/ziutek/mymysql v1.5.4 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.0.0-20240812153829-bb9ac54eca05 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.35.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.60.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.0.0-20240805233418-127d068751eb // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.4.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.35.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.28.0 // indirect
	go.opentelemetry.io/otel/log v0.5.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.5.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/exp v0.0.0-20250305212735-054e65f0b394 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250505200425-f936aa4a68b2 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)
