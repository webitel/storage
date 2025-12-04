module github.com/webitel/storage

go 1.24.1

require (
	cloud.google.com/go/speech v1.26.0
	cloud.google.com/go/storage v1.54.0
	cloud.google.com/go/texttospeech v1.11.0
	github.com/BoRuDar/configuration/v4 v4.2.2
	github.com/aws/aws-sdk-go v1.55.8
	github.com/dutchcoders/go-clamd v0.0.0-20170520113014-b970184f4d9e
	github.com/go-gorp/gorp v2.2.0+incompatible
	github.com/golang/protobuf v1.5.4
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.3
	github.com/h2non/filetype v1.1.3
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/jmoiron/sqlx v1.4.0
	github.com/juju/ratelimit v1.0.2
	github.com/lib/pq v1.10.9
	github.com/makiuchi-d/gozxing v0.1.1
	github.com/pborman/uuid v1.2.1
	github.com/pkg/errors v0.9.1
	github.com/rabbitmq/amqp091-go v1.10.0
	github.com/robfig/cron v1.2.0
	github.com/webitel/engine/pkg/discovery v0.0.0-20250925094226-59b6a9641d79
	github.com/webitel/engine/pkg/presign v0.0.0-20250508102602-4109efdb5e30
	github.com/webitel/engine/pkg/wbt v0.0.0-20250919085327-014f1a899fd3
	github.com/webitel/engine/pkg/werror v0.0.0-20250508121332-6ae1563235d8
	github.com/webitel/webitel-go-kit/infra/logger_client v0.0.0-20250605113615-1ee94622655d
	github.com/webitel/webitel-go-kit/infra/otel v0.0.0-20250605113615-1ee94622655d
	github.com/webitel/webitel-go-kit/infra/pubsub/rabbitmq v0.0.0-20250605113615-1ee94622655d
	github.com/webitel/webitel-go-kit/pkg/watcher v0.0.0-20250605113615-1ee94622655d
	github.com/webitel/wlog v0.0.0-20250325101442-de4f125c1ec7
	go.opentelemetry.io/otel v1.36.0
	go.opentelemetry.io/otel/sdk v1.36.0
	golang.org/x/crypto v0.38.0
	golang.org/x/image v0.12.0
	golang.org/x/sync v0.14.0
	google.golang.org/api v0.233.0
	google.golang.org/genproto v0.0.0-20250303144028-a0af3efb3deb
	google.golang.org/genproto/googleapis/api v0.0.0-20250519155744-55703ea1f237
	google.golang.org/grpc v1.72.2
	google.golang.org/protobuf v1.36.10
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
	github.com/cenkalti/backoff/v5 v5.0.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cncf/xds/go v0.0.0-20250121191232-2f005788dc42 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.32.4 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
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
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/miekg/dns v1.1.43 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/poy/onpar v1.1.2 // indirect
	github.com/spiffe/go-spiffe/v2 v2.5.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/zeebo/errs v1.4.0 // indirect
	github.com/ziutek/mymysql v1.5.4 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.11.0 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.35.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.60.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.12.2 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.12.2 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.36.0 // indirect
	go.opentelemetry.io/otel/log v0.12.2 // indirect
	go.opentelemetry.io/otel/log/logtest v0.0.0-20250605073931-9fabec1f4432 // indirect
	go.opentelemetry.io/otel/metric v1.36.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.12.2 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.36.0 // indirect
	go.opentelemetry.io/otel/trace v1.36.0 // indirect
	go.opentelemetry.io/proto/otlp v1.6.0 // indirect
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
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250519155744-55703ea1f237 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)
