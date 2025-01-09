package app

import (
	"context"
	"fmt"
	otelsdk "github.com/webitel/webitel-go-kit/otel/sdk"
	"go.opentelemetry.io/otel/sdk/resource"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/webitel/engine/auth_manager"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/engine/presign"
	"github.com/webitel/storage/interfaces"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/store"
	"github.com/webitel/storage/store/sqlstore"
	"github.com/webitel/storage/utils"
	"github.com/webitel/wlog"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	// -------------------- plugin(s) -------------------- //
	_ "github.com/webitel/webitel-go-kit/otel/sdk/log/otlp"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/log/stdout"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/metric/otlp"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/metric/stdout"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/trace/otlp"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/trace/stdout"
)

const filePolicyExpire = 5

type App struct {
	id          *string
	Srv         *Server
	InternalSrv *Server
	cluster     *cluster
	GrpcServer  *GrpcServer

	MediaFileStore   utils.FileBackend
	FileCache        utils.FileBackend
	DefaultFileStore utils.FileBackend

	fileBackendCache *utils.Cache
	sttProfilesCache *utils.Cache
	jobCallback      *utils.Cache

	Store store.Store

	Log        *wlog.Logger
	configFile string
	config     atomic.Value
	newStore   func() store.Store
	//Jobs       *jobs.JobServer

	sessionManager auth_manager.AuthManager
	Uploader       interfaces.UploadRecordingsFilesInterface
	Synchronizer   interfaces.SynchronizerFilesInterface
	filePolicies   *DomainFilePolicy

	preSigned presign.PreSign

	upTime time.Time

	thumbnailSettings model.ThumbnailSettings

	ctx              context.Context
	otelShutdownFunc otelsdk.ShutdownFunc
}

func New(options ...string) (outApp *App, outErr error) {
	rootRouter := mux.NewRouter()
	internalRootRouter := mux.NewRouter()

	app := &App{
		upTime: time.Now(),
		Srv: &Server{
			RootRouter: rootRouter,
		},
		InternalSrv: &Server{
			RootRouter: internalRootRouter,
		},
		fileBackendCache: utils.NewLru(model.BackendCacheSize),
		sttProfilesCache: utils.NewLru(model.SttCacheSize),
		jobCallback:      utils.NewLru(model.JobCacheSize),
		ctx:              context.Background(),
	}
	app.Srv.Router = app.Srv.RootRouter.PathPrefix("/").Subrouter()
	app.InternalSrv.Router = app.InternalSrv.RootRouter.PathPrefix("/").Subrouter()

	app.filePolicies = &DomainFilePolicy{
		app:      app,
		policies: utils.NewLruWithParams(100, "domain policies", filePolicyExpire, ""),
	}

	defer func() {
		if outErr != nil {
			app.Shutdown()
		}
	}()

	if err := app.LoadConfig(app.configFile); err != nil {
		return nil, err
	}

	config := app.Config()

	app.thumbnailSettings = config.Thumbnail

	if utils.T == nil {
		if err := utils.TranslationsPreInit(app.Config().TranslationsDirectory); err != nil {
			return nil, errors.Wrapf(err, "unable to load translation files")
		}
	}

	engine.AppErrorInit(utils.T)

	logConfig := &wlog.LoggerConfiguration{
		EnableConsole: config.Log.Console,
		ConsoleJson:   false,
		ConsoleLevel:  config.Log.Lvl,
	}

	if config.Log.File != "" {
		logConfig.FileLocation = config.Log.File
		logConfig.EnableFile = true
		logConfig.FileJson = true
		logConfig.FileLevel = config.Log.Lvl
	}

	if config.Log.Otel {
		// TODO
		var err error
		logConfig.EnableExport = true
		app.otelShutdownFunc, err = otelsdk.Configure(
			app.ctx,
			otelsdk.WithResource(resource.NewSchemaless(
				semconv.ServiceName(model.APP_SERVICE_NAME),
				semconv.ServiceVersion(model.CurrentVersion),
				semconv.ServiceInstanceID(*app.id),
				semconv.ServiceNamespace("webitel"),
			)),
		)
		if err != nil {
			return nil, err
		}
	}
	app.Log = wlog.NewLogger(logConfig)

	wlog.RedirectStdLog(app.Log)
	wlog.InitGlobalLogger(app.Log)

	if err := utils.InitTranslations(app.Config().LocalizationSettings); err != nil {
		return nil, errors.Wrapf(err, "unable to load translation files")
	}

	if preSign, err := presign.NewPreSigned(app.Config().PreSignedCertificateLocation); err != nil {
		return nil, errors.Wrapf(err, "unable to load certificate file")
	} else {
		app.preSigned = preSign
	}

	if err := app.initLocalFileStores(); err != nil {
		return nil, err
	}

	wlog.Info("Server is initializing...")

	app.cluster = NewCluster(app)

	if app.newStore == nil {
		app.newStore = func() store.Store {
			return store.NewLayeredStore(sqlstore.NewSqlSupplier(app.Config().SqlSettings))
		}
	}

	app.Srv.Store = app.newStore()
	app.Store = app.Srv.Store

	app.GrpcServer = NewGrpcServer(app.Config().ServerSettings)

	if outErr = app.cluster.Start(); outErr != nil {
		return nil, outErr
	}

	app.sessionManager = auth_manager.NewAuthManager(model.SESSION_CACHE_SIZE, model.SESSION_CACHE_TIME,
		app.cluster.discovery, app.Log)
	if err := app.sessionManager.Start(); err != nil {
		return nil, err
	}

	app.Srv.Router.NotFoundHandler = http.HandlerFunc(app.Handle404)
	app.InternalSrv.Router.NotFoundHandler = http.HandlerFunc(app.Handle404)

	app.initUploader()
	app.initSynchronizer()
	return app, outErr
}

func (app *App) initLocalFileStores() engine.AppError {
	var appErr engine.AppError
	mediaSettings := app.Config().MediaFileStoreSettings
	fileSettings := app.Config().DefaultFileStore

	if app.FileCache, appErr = utils.NewBackendStore(&model.FileBackendProfile{
		Name:       "Internal file cache",
		Type:       model.FileDriverLocal,
		Properties: model.StringInterface{"directory": model.CacheDir, "path_pattern": ""},
	}); appErr != nil {
		return appErr
	}

	if app.MediaFileStore, appErr = utils.NewBackendStore(&model.FileBackendProfile{
		Name:       "Media store",
		Type:       model.FileDriverLocal,
		Properties: model.StringInterface{"directory": mediaSettings.Directory, "path_pattern": mediaSettings.PathPattern},
	}); appErr != nil {
		return appErr
	}

	if fileSettings != nil {
		if app.DefaultFileStore, appErr = utils.NewBackendStore(&model.FileBackendProfile{
			Name:       "Default record file store",
			Type:       model.StorageBackendTypeFromString(fileSettings.Type),
			ExpireDay:  fileSettings.ExpireDay,
			Properties: fileSettings.Props,
		}); appErr != nil {
			return appErr
		}
	}

	return nil
}
func (app *App) UseDefaultStore() bool {
	return app.DefaultFileStore != nil
}

func (app *App) Shutdown() {
	wlog.Info("Stopping Server...")

	if app.Srv.Server != nil {
		app.Srv.Server.Close()
	}

	if app.InternalSrv.Server != nil {
		app.InternalSrv.Server.Close()
	}

	if app.cluster != nil {
		app.cluster.Stop()
	}

	if app.otelShutdownFunc != nil {
		app.otelShutdownFunc(app.ctx)
	}
}

func (a *App) Handle404(w http.ResponseWriter, r *http.Request) {
	err := engine.NewNotFoundError("api.context.404.app_error", r.URL.String())
	ip := utils.GetIpAddress(r)
	a.Log.Debug(fmt.Sprintf("%v: code=404 ip=%v", r.URL.Path, ip),
		wlog.String("ip", ip),
	)
	utils.RenderWebAppError(a.Config(), w, r, err)
}

func (a *App) GetInstanceId() string {
	return *a.id
}

func (a *App) initUploader() {
	if uploadRecordingsFilesInterface != nil {
		a.Uploader = uploadRecordingsFilesInterface(a)
	}
}

func (a *App) initSynchronizer() {
	if synchronizerFilesInterface != nil {
		a.Synchronizer = synchronizerFilesInterface(a)
	}
}

var uploadRecordingsFilesInterface func(*App) interfaces.UploadRecordingsFilesInterface

func RegisterUploader(f func(*App) interfaces.UploadRecordingsFilesInterface) {
	uploadRecordingsFilesInterface = f
}

var synchronizerFilesInterface func(*App) interfaces.SynchronizerFilesInterface

func RegisterSynchronizer(f func(*App) interfaces.SynchronizerFilesInterface) {
	synchronizerFilesInterface = f
}
