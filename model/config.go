package model

import (
	engine "github.com/webitel/engine/model"
)

const (
	DEFAULT_LOCALE = "en"

	DATABASE_DRIVER_POSTGRES = "postgres"
)

type LocalizationSettings struct {
	DefaultServerLocale *string
	DefaultClientLocale *string
	AvailableLocales    *string
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
	ListenAddress         *string
	ListenInternalAddress *string
	PublicHost            string
}

type SqlSettings struct {
	DriverName                  *string
	DataSource                  *string
	DataSourceReplicas          []string
	DataSourceSearchReplicas    []string
	MaxIdleConns                *int
	ConnMaxLifetimeMilliseconds *int
	MaxOpenConns                *int
	Trace                       bool
	AtRestEncryptKey            string
	QueryTimeout                *int
}

type NoSqlSettings struct {
	Host  *string
	Trace bool
}

type Config struct {
	TranslationsDirectory        string
	NodeName                     string
	IsDev                        bool
	PreSignedCertificateLocation string
	PreSignedTimeout             int64
	DiscoverySettings            DiscoverySettings
	LocalizationSettings         LocalizationSettings
	ServiceSettings              ServiceSettings
	SqlSettings                  SqlSettings
	MediaFileStoreSettings       MediaFileStoreSettings
	DefaultFileStore             *DefaultFileStore
	ServerSettings               ServerSettings
	ProxyUploadUrl               *string
}

type DiscoverySettings struct {
	Url string
}

type ServerSettings struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
	Network string `json:"network"`
}

type MediaFileStoreSettings struct {
	MaxSizeByte       *int
	AllowMime         []string
	Directory         *string
	PathPattern       *string
	MaxUploadFileSize int64
}

type DefaultFileStore struct {
	Type      string
	ExpireDay int
	Props     StringInterface
}

func (c *Config) IsValid() engine.AppError {

	if c.MediaFileStoreSettings.Directory == nil || len(*c.MediaFileStoreSettings.Directory) == 0 {
		return engine.NewInternalError("model.config.is_valid.media_store_directory.app_error", "")
	}
	return nil
}
