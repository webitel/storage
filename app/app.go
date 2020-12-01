package app

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/webitel/engine/auth_manager"
	presign "github.com/webitel/engine/presign"
	"github.com/webitel/storage/broker"
	"github.com/webitel/storage/interfaces"
	"github.com/webitel/storage/jobs"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/store"
	"github.com/webitel/storage/store/sqlstore"
	"github.com/webitel/storage/utils"
	"github.com/webitel/wlog"
	"net/http"
	"sync/atomic"
	"time"
)

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

	Store  store.Store
	Broker broker.Broker

	Log        *wlog.Logger
	configFile string
	config     atomic.Value
	newStore   func() store.Store
	Jobs       *jobs.JobServer

	sessionManager auth_manager.AuthManager
	Uploader       interfaces.UploadRecordingsFilesInterface

	preSigned presign.PreSign

	upTime time.Time
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
		fileBackendCache: utils.NewLru(model.ACTIVE_BACKEND_CACHE_SIZE),
	}
	app.Srv.Router = app.Srv.RootRouter.PathPrefix("/").Subrouter()
	app.InternalSrv.Router = app.InternalSrv.RootRouter.PathPrefix("/").Subrouter()

	defer func() {
		if outErr != nil {
			app.Shutdown()
		}
	}()

	if err := app.LoadConfig(app.configFile); err != nil {
		return nil, err
	}

	if utils.T == nil {
		if err := utils.TranslationsPreInit(app.Config().TranslationsDirectory); err != nil {
			return nil, errors.Wrapf(err, "unable to load translation files")
		}
	}

	model.AppErrorInit(utils.T)

	app.Log = wlog.NewLogger(&wlog.LoggerConfiguration{
		EnableConsole: true,
		ConsoleLevel:  wlog.LevelDebug,
	})

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

	app.initLocalFileStores()

	wlog.Info("Server is initializing...")

	app.cluster = NewCluster(app)

	//app.Broker = broker.NewLayeredBroker(amqp.NewBrokerSupplier(app.Config().BrokerSettings), app)

	if app.newStore == nil {
		app.newStore = func() store.Store {
			return store.NewLayeredStore(sqlstore.NewSqlSupplier(app.Config().SqlSettings), store.NewElasticSupplier(app.Config().NoSqlSettings))
		}
	}

	app.Srv.Store = app.newStore()
	app.Store = app.Srv.Store

	app.GrpcServer = NewGrpcServer(app.Config().ServerSettings)

	if outErr = app.cluster.Start(); outErr != nil {
		return nil, outErr
	}

	app.sessionManager = auth_manager.NewAuthManager(model.SESSION_CACHE_SIZE, model.SESSION_CACHE_TIME, app.cluster.discovery)
	if err := app.sessionManager.Start(); err != nil {
		return nil, err
	}

	app.Srv.Router.NotFoundHandler = http.HandlerFunc(app.Handle404)
	app.InternalSrv.Router.NotFoundHandler = http.HandlerFunc(app.Handle404)

	app.initJobs()
	app.initUploader()
	return app, outErr
}

func (app *App) initLocalFileStores() *model.AppError {
	var appErr *model.AppError
	mediaSettings := app.Config().MediaFileStoreSettings
	fileSettings := app.Config().DefaultFileStore

	if app.FileCache, appErr = utils.NewBackendStore(&model.FileBackendProfile{
		Name:       "Internal file cache",
		Type:       model.FileDriverLocal,
		Properties: model.StringInterface{"directory": model.CACHE_DIR, "path_pattern": ""},
	}); appErr != nil {
		return appErr
	}

	if app.MediaFileStore, appErr = utils.NewBackendStore(&model.FileBackendProfile{
		Name:       "Media store",
		Type:       model.FileDriverLocal,
		Properties: model.StringInterface{"directory": *mediaSettings.Directory, "path_pattern": *mediaSettings.PathPattern},
	}); appErr != nil {
		return appErr
	}

	if fileSettings != nil {
		if app.DefaultFileStore, appErr = utils.NewBackendStore(&model.FileBackendProfile{
			Name:       "Internal record file store",
			Type:       model.StorageBackendTypeFromString(fileSettings.Type),
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
}

func (a *App) Handle404(w http.ResponseWriter, r *http.Request) {
	err := model.NewAppError("Handle404", "api.context.404.app_error", nil, r.URL.String(), http.StatusNotFound)
	wlog.Debug(fmt.Sprintf("%v: code=404 ip=%v", r.URL.Path, utils.GetIpAddress(r)))
	utils.RenderWebAppError(a.Config(), w, r, err)
}

func (a *App) GetInstanceId() string {
	return *a.id
}

func (a *App) initJobs() {
	a.Jobs = jobs.NewJobServer(a, a.Store)

	if syncFilesJobInterface != nil {
		a.Jobs.AddMiddleware(model.JOB_TYPE_SYNC_FILES, syncFilesJobInterface(a))
	}
}

func (a *App) initUploader() {
	if uploadRecordingsFilesInterface != nil {
		a.Uploader = uploadRecordingsFilesInterface(a)
	}
}

var uploadRecordingsFilesInterface func(*App) interfaces.UploadRecordingsFilesInterface

func RegisterUploader(f func(*App) interfaces.UploadRecordingsFilesInterface) {
	uploadRecordingsFilesInterface = f
}

var syncFilesJobInterface func(*App) interfaces.SyncFilesJobInterface

func RegisterSyncFilesJobInterface(f func(*App) interfaces.SyncFilesJobInterface) {
	syncFilesJobInterface = f
}
