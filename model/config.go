package model

import (
	engine "github.com/webitel/engine/model"
	"time"
)

const (
	DEFAULT_LOCALE = "en"

	DATABASE_DRIVER_POSTGRES = "postgres"
)

type LocalizationSettings struct {
	DefaultServerLocale *string `json:"default_server_locale" default:"en"`
	DefaultClientLocale *string `json:"default_client_locale" default:"en"`
	AvailableLocales    *string `json:"available_locales" default:"en"`
}

func (s *LocalizationSettings) SetDefaults() {
	if s.DefaultServerLocale == nil {
		s.DefaultServerLocale = NewString(DEFAULT_LOCALE)
	}

	if s.DefaultClientLocale == nil {
		s.DefaultClientLocale = NewString(DEFAULT_LOCALE)
	}

	if s.AvailableLocales == nil {
		s.AvailableLocales = NewString("")
	}
}

type ServiceSettings struct {
	ListenAddress         string `json:"public_address" flag:"public_address|:10023|Public server address" env:"PUBLIC_ADDRESS"`
	ListenInternalAddress string `json:"internal_address" flag:"internal_address|:10021|Internal server address" env:"INTERNAL_ADDRESS"`
	PublicHost            string `json:"public_host" flag:"public_host|https://dev.webitel.com/|Public host" env:"PUBLIC_HOST"`
}

type SqlSettings struct {
	DriverName                  *string  `json:"driver_name" flag:"sql_driver_name|postgres|" env:"SQL_DRIVER_NAME"`
	DataSource                  *string  `json:"data_source" flag:"data_source|postgres://opensips:webitel@postgres:5432/webitel?fallback_application_name=engine&sslmode=disable&connect_timeout=10&search_path=call_center|Data source" env:"DATA_SOURCE"`
	DataSourceReplicas          []string `json:"data_source_replicas" flag:"sql_data_source_replicas" default:"" env:"SQL_DATA_SOURCE_REPLICAS"`
	MaxIdleConns                *int     `json:"max_idle_conns" flag:"sql_max_idle_conns|5|Maximum idle connections" env:"SQL_MAX_IDLE_CONNS"`
	MaxOpenConns                *int     `json:"max_open_conns" flag:"sql_max_open_conns|5|Maximum open connections" env:"SQL_MAX_OPEN_CONNS"`
	ConnMaxLifetimeMilliseconds *int     `json:"conn_max_lifetime_milliseconds" flag:"sql_conn_max_lifetime_milliseconds|300000|Connection maximum lifetime milliseconds" env:"SQL_LIFETIME_MILLISECONDS"`
	Trace                       bool     `json:"trace" flag:"sql_trace|false|Trace SQL" env:"SQL_TRACE"`
	Log                         bool     `json:"log" flag:"sql_log|false|Log SQL" env:"SQL_LOG"`
	QueryTimeout                *int     `json:"query_timeout" flag:"sql_query_timeout|10|Sql query timeout seconds" env:"QUERY_TIMEOUT"`
}

type NoSqlSettings struct {
	Host  *string
	Trace bool
}

type Config struct {
	TranslationsDirectory        string                 `json:"translations_directory" flag:"translations_directory|i18n|Translations directory" env:"TRANSLATION_DIRECTORY"`
	NodeName                     string                 `flag:"id|1|Service id" json:"id" env:"ID"`
	IsDev                        bool                   `json:"dev" flag:"dev|false|Dev mode" env:"DEV"`
	PreSignedCertificateLocation string                 `json:"presigned_cert" flag:"presigned_cert|/opt/storage/key.pem|Location to pre signed certificate" env:"PRESIGNED_CERT"`
	PreSignedTimeout             int64                  `json:"presigned_timeout" flag:"presigned_timeout|900000|Pre signed timeout" env:"PRESIGNED_TIMEOUT"`
	DiscoverySettings            DiscoverySettings      `json:"discovery_settings"`
	LocalizationSettings         LocalizationSettings   `json:"localization_settings"`
	ServiceSettings              ServiceSettings        `json:"service_settings"`
	SqlSettings                  SqlSettings            `json:"sql_settings"`
	MediaFileStoreSettings       MediaFileStoreSettings `json:"media_file_store_settings"`

	DefaultFileStore   *DefaultFileStore `json:"default_file_store"`
	ServerSettings     ServerSettings    `json:"server_settings"`
	ProxyUploadUrl     string            `json:"proxy_upload" flag:"proxy_upload||Proxy upload url" env:"PROXY_UPLOAD"`
	MaxSafeUploadSleep time.Duration     `json:"safe_upload_max_sleep" flag:"safe_upload_max_sleep|60sec|Maximum upload second sleep process" env:"SAFE_UPLOAD_MAX_SLEEP"`
	Thumbnail          ThumbnailSettings `json:"thumbnail"`
	Log                LogSettings       `json:"log"`
	TtsEndpoint        string            `json:"tts_endpoint" flag:"wbt_tts_endpoint||Offline TTS endpoint" env:"WBT_TTS_ENDPOINT"`
}

type LogSettings struct {
	Lvl     string `json:"lvl" flag:"log_lvl|debug|Log level" env:"LOG_LVL"`
	Json    bool   `json:"json" flag:"log_json|false|Log format JSON" env:"LOG_JSON"`
	Otel    bool   `json:"otel" flag:"log_otel|false|Log OTEL" env:"LOG_OTEL"`
	File    string `json:"file" flag:"log_file||Log file directory" env:"LOG_FILE"`
	Console bool   `json:"console" flag:"log_console|false|Log console" env:"LOG_CONSOLE"`
}

type ThumbnailSettings struct {
	ForceEnabled bool   `json:"force_enabled" flag:"thumbnail_force_enabled|0|Create thumbnail by default" env:"THUMBNAIL_FORCE_ENABLE"`
	DefaultScale string `json:"default_scale" flag:"thumbnail_default_scale||Default scale for thumbnail" env:"THUMBNAIL_DEFAULT_SCALE"`
}

type DiscoverySettings struct {
	Url string `json:"url" flag:"consul|172.0.0.1:8500|Host to consul" env:"CONSUL"`
}

type ServerSettings struct {
	Address string `json:"address" flag:"grpc_addr||GRPC host" env:"GRPC_ADDR"`
	Port    int    `json:"port" flag:"grpc_port|0|GRPC port" env:"GRPC_PORT"`
	Network string `json:"network" flag:"grpc_network|tcp|GRPC network" env:"GRPC_NETWORK"`
}

type MediaFileStoreSettings struct {
	Directory               string `json:"media_directory" flag:"media_directory|/data|Media file directory" env:"MEDIA_DIRECTORY"`
	PathPattern             string `json:"media_store_pattern" flag:"media_store_pattern|$DOMAIN|Media store pattern" env:"MEDIA_STORE_PATTERN"`
	MaxUploadFileSizeString string `json:"max_upload_file_size" flag:"max_upload_file_size|20MB|Media file directory" env:"MAX_UPLOAD_FILE_SIZE"`
	MaxUploadFileSize       int64  `json:"-"`
}

type DefaultFileStore struct {
	Type        string          `json:"type" flag:"file_store_type||Default file store type" env:"FILE_STORE_TYPE"`
	ExpireDay   int             `json:"expire_day" flag:"file_store_expire_day|0|Default file expire day (0 - never delete)" env:"FILE_STORE_EXPIRE_DAY"`
	PropsString string          `json:"-" flag:"file_store_props||Default file store props" env:"FILE_STORE_PROPS"`
	Props       StringInterface `json:"props"`
}

func (c *Config) IsValid() engine.AppError {

	if c.MediaFileStoreSettings.Directory == "" {
		return engine.NewInternalError("model.config.is_valid.media_store_directory.app_error", "")
	}
	return nil
}
