package app

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/webitel/storage/tts"

	"github.com/webitel/storage/utils"

	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
)

var (
	appId                 = flag.String("id", "1", "Service id")
	translationsDirectory = flag.String("translations_directory", "i18n", "Translations directory")
	consulHost            = flag.String("consul", "consul:8500", "Host to consul")
	dataSource            = flag.String("data_source", "postgres://opensips:webitel@postgres:5432/webitel?fallback_application_name=storage&sslmode=disable&connect_timeout=10&search_path=storage", "Data source")
	grpcServerPort        = flag.Int("grpc_port", 0, "GRPC port")
	grpcServerAddr        = flag.String("grpc_addr", "", "GRPC host")
	dev                   = flag.Bool("dev", false, "enable dev mode")
	internalServerAddress = flag.String("internal_address", ":10021", "Internal server address")
	publicServerAddress   = flag.String("public_address", ":10023", "Public server address")
	mediaDirectory        = flag.String("media_directory", "/data", "Media file directory")
	mediaStorePattern     = flag.String("media_store_pattern", "$DOMAIN", "Media store pattern")

	defaultFileStoreType  = flag.String("file_store_type", "", "Default file store type")
	defaultFileStoreProps = flag.String("file_store_props", "", "Default file store props")
	defaultFileExpireDay  = flag.Int("file_store_expire_day", 0, "Default file expire day (0 - never delete)")
	allowMediaMime        = flag.String("allow_media", "", "Allow upload media mime type")
	maxUploadFileSize     = flag.String("max_upload_file_size", "20MB", "Maximum upload file size")

	presignedCertFile = flag.String("presigned_cert", "/opt/storage/key.pem", "Location to pre signed certificate")
	presignedTimeout  = flag.Int64("presigned_timeout", 1000*60*15, "Pre signed timeout")

	proxyUpload = flag.String("proxy_upload", "", "Proxy upload url")
	publicHost  = flag.String("public_host", "https://dev.webitel.com/", "Public host")

	wbtTTSEndpoint = flag.String("wbt_tts_endpoint", "", "Offline TTS endpoint")
)

func loadConfig(fileName string) (*model.Config, engine.AppError) {
	flag.Parse()
	var mimeTypes []string
	if *allowMediaMime != "" {
		mimeTypes = strings.Split(*allowMediaMime, ",")
	}

	maxUploadSizeInByte, err := utils.FromHumanSize(*maxUploadFileSize)
	if err != nil {
		panic(err.Error())
	}

	cfg := &model.Config{
		TranslationsDirectory:        *translationsDirectory,
		PreSignedCertificateLocation: *presignedCertFile,
		PreSignedTimeout:             *presignedTimeout,
		NodeName:                     fmt.Sprintf("%s-%s", model.APP_SERVICE_NAME, *appId),
		IsDev:                        *dev,
		LocalizationSettings: model.LocalizationSettings{
			DefaultClientLocale: model.NewString(model.DEFAULT_LOCALE),
			DefaultServerLocale: model.NewString(model.DEFAULT_LOCALE),
			AvailableLocales:    model.NewString(model.DEFAULT_LOCALE),
		},
		ServiceSettings: model.ServiceSettings{
			ListenAddress:         publicServerAddress,
			ListenInternalAddress: internalServerAddress,
			PublicHost:            *publicHost,
		},
		MediaFileStoreSettings: model.MediaFileStoreSettings{
			MaxSizeByte:       model.NewInt(100 * 1000000),
			Directory:         mediaDirectory,
			PathPattern:       mediaStorePattern,
			AllowMime:         mimeTypes,
			MaxUploadFileSize: maxUploadSizeInByte,
		},
		SqlSettings: model.SqlSettings{
			DriverName:                  model.NewString("postgres"),
			DataSource:                  dataSource,
			MaxIdleConns:                model.NewInt(5),
			MaxOpenConns:                model.NewInt(5),
			ConnMaxLifetimeMilliseconds: model.NewInt(3600000),
			Trace:                       false,
		},
		DiscoverySettings: model.DiscoverySettings{
			Url: *consulHost,
		},
		ServerSettings: model.ServerSettings{
			Address: *grpcServerAddr,
			Port:    *grpcServerPort,
			Network: "tcp",
		},
	}

	if proxyUpload != nil && *proxyUpload != "" {
		cfg.ProxyUploadUrl = proxyUpload
	}

	if defaultFileStoreType != nil && *defaultFileStoreType != "" {
		cfg.DefaultFileStore = &model.DefaultFileStore{
			Type:      *defaultFileStoreType,
			ExpireDay: *defaultFileExpireDay,
		}

		if defaultFileStoreProps != nil {
			err := json.Unmarshal([]byte(*defaultFileStoreProps), &cfg.DefaultFileStore.Props)
			if err != nil {
				panic(err)
			}
		}
	}

	if strings.HasSuffix(cfg.ServiceSettings.PublicHost, "/") {
		cfg.ServiceSettings.PublicHost = cfg.ServiceSettings.PublicHost[:len(cfg.ServiceSettings.PublicHost)-1]
	}

	if wbtTTSEndpoint != nil && len(*wbtTTSEndpoint) != 0 {
		tts.SetWbtTTSEndpoint(*wbtTTSEndpoint) // TODO
	}

	return cfg, nil
}

func (a *App) Config() *model.Config {
	if cfg := a.config.Load(); cfg != nil {
		return cfg.(*model.Config)
	}
	return &model.Config{}
}

func (a *App) LoadConfig(configFile string) engine.AppError {
	cfg, err := loadConfig(configFile)
	if err != nil {
		return err
	}

	if err = cfg.IsValid(); err != nil {
		return err
	}
	a.configFile = configFile
	a.id = &cfg.NodeName

	a.config.Store(cfg)
	return nil
}
