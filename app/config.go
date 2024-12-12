package app

import (
	"encoding/json"
	"github.com/BoRuDar/configuration/v4"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/tts"
	"github.com/webitel/storage/utils"
)

func loadConfig(fileName string) (*model.Config, engine.AppError) {
	var config model.Config
	configurator := configuration.New(
		&config,
		configuration.NewEnvProvider(),
		configuration.NewFlagProvider(),
		configuration.NewDefaultProvider(),
	).SetOptions(configuration.OnFailFnOpt(func(err error) {
		//log.Println(err)
	}))

	if err := configurator.InitValues(); err != nil {
		//return nil, err
	}

	maxUploadSizeInByte, err := utils.FromHumanSize(config.MediaFileStoreSettings.MaxUploadFileSizeString)
	if err != nil {
		panic(err.Error())
	}
	config.MediaFileStoreSettings.MaxUploadFileSize = maxUploadSizeInByte

	if config.DefaultFileStore != nil && config.DefaultFileStore.Type != "" {
		if config.DefaultFileStore.PropsString != "" {
			err = json.Unmarshal([]byte(config.DefaultFileStore.PropsString), &config.DefaultFileStore.Props)
			if err != nil {
				panic(err)
			}
		}
	} else {
		config.DefaultFileStore = nil
	}

	if config.TtsEndpoint != "" {
		tts.SetWbtTTSEndpoint(config.TtsEndpoint)
	}

	if !config.Log.Console && !config.Log.Otel && len(config.Log.File) == 0 {
		config.Log.Console = true
	}

	return &config, nil
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
