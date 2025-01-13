package store

import (
	"context"
	"time"

	"github.com/webitel/engine/auth_manager"

	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
)

type StoreResult struct {
	Data interface{}
	Err  engine.AppError
}

type StoreChannel chan StoreResult

func Do(f func(result *StoreResult)) StoreChannel {
	storeChannel := make(StoreChannel, 1)
	go func() {
		result := StoreResult{}
		f(&result)
		storeChannel <- result
		close(storeChannel)
	}()
	return storeChannel
}

func Must(sc StoreChannel) interface{} {
	r := <-sc
	if r.Err != nil {

		time.Sleep(time.Second)
		panic(r.Err)
	}

	return r.Data
}

type Store interface {
	StoreData
}

type StoreData interface {
	UploadJob() UploadJobStore
	FileBackendProfile() FileBackendProfileStore
	File() FileStore
	Job() JobStore
	MediaFile() MediaFileStore
	Schedule() ScheduleStore
	SyncFile() SyncFileStore
	CognitiveProfile() CognitiveProfileStore
	TranscriptFile() TranscriptFileStore
	ImportTemplate() ImportTemplateStore
	FilePolicies() FilePoliciesStore
	SystemSettings() SystemSettingsStore
}

type UploadJobStore interface {
	Create(job *model.JobUploadFile) (*model.JobUploadFile, engine.AppError)
	//Save(job *model.JobUploadFile) StoreChannel
	GetAllPageByInstance(limit int, instance string) StoreChannel
	UpdateWithProfile(limit int, instance string, betweenAttemptSec int64, defStore bool) StoreChannel
	SetStateError(id int, errMsg string) StoreChannel
}

type SyncFileStore interface {
	FetchJobs(limit int) ([]*model.SyncJob, engine.AppError)
	SetRemoveJobs(localExpDay int) engine.AppError
	Clean(jobId int64) engine.AppError
	Remove(jobId int64) engine.AppError

	RemoveErrors() engine.AppError
	SetError(jobId int64, e error) engine.AppError
}

type FileBackendProfileStore interface {
	CheckAccess(domainId, id int64, groups []int, access auth_manager.PermissionAccess) (bool, engine.AppError)
	Create(profile *model.FileBackendProfile) (*model.FileBackendProfile, engine.AppError)
	GetAllPage(domainId int64, req *model.SearchFileBackendProfile) ([]*model.FileBackendProfile, engine.AppError)
	GetAllPageByGroups(domainId int64, groups []int, search *model.SearchFileBackendProfile) ([]*model.FileBackendProfile, engine.AppError)
	Get(id, domainId int64) (*model.FileBackendProfile, engine.AppError)
	GetById(id int) (*model.FileBackendProfile, engine.AppError)
	Update(profile *model.FileBackendProfile) (*model.FileBackendProfile, engine.AppError)
	Delete(domainId, id int64) engine.AppError
	GetSyncTime(domainId int64, id int) (*model.FileBackendProfileSync, engine.AppError)
}

type FileStore interface {
	GetAllPage(ctx context.Context, domainId int64, search *model.SearchFile) ([]*model.File, engine.AppError)
	Create(file *model.File) StoreChannel
	GetFileWithProfile(domainId, id int64) (*model.FileWithProfile, engine.AppError)
	GetFileByUuidWithProfile(domainId int64, uuid string) (*model.FileWithProfile, engine.AppError)
	MarkRemove(domainId int64, ids []int64) engine.AppError

	MoveFromJob(jobId int64, profileId *int, properties model.StringInterface) StoreChannel
	CheckCallRecordPermissions(ctx context.Context, fileId int, currentUserId int64, domainId int64, groups []int) (bool, engine.AppError)
}

type MediaFileStore interface {
	Create(file *model.MediaFile) (*model.MediaFile, engine.AppError)
	GetAllPage(domainId int64, search *model.SearchMediaFile) ([]*model.MediaFile, engine.AppError)
	Get(domainId int64, id int) (*model.MediaFile, engine.AppError)
	Delete(domainId, id int64) engine.AppError

	Save(file *model.MediaFile) StoreChannel
	GetAllByDomain(domain string, offset, limit int) StoreChannel
	GetCountByDomain(domain string) StoreChannel
	GetByName(name, domain string) StoreChannel
	DeleteByName(name, domain string) StoreChannel
	DeleteById(id int64) StoreChannel
}

type ScheduleStore interface {
	GetAllEnablePage(limit, offset int) StoreChannel
	GetAllWithNoJobs(limit, offset int) StoreChannel
	GetAllPageByType(typeName string) StoreChannel
}

type JobStore interface {
	Save(job *model.Job) (*model.Job, engine.AppError)
	UpdateOptimistically(job *model.Job, currentStatus string) StoreChannel
	UpdateStatus(id string, status string) StoreChannel
	UpdateStatusOptimistically(id string, currentStatus string, newStatus string) StoreChannel
	Get(id string) StoreChannel
	GetAllPage(offset int, limit int) StoreChannel
	GetAllByType(jobType string) StoreChannel
	GetAllByTypePage(jobType string, offset int, limit int) StoreChannel
	GetAllByStatus(status string) StoreChannel
	GetAllByStatusAndLessScheduleTime(status string, t int64) StoreChannel
	GetNewestJobByStatusAndType(status string, jobType string) StoreChannel
	GetCountByStatusAndType(status string, jobType string) StoreChannel
	Delete(id string) StoreChannel
}

type CognitiveProfileStore interface {
	CheckAccess(domainId, id int64, groups []int, access auth_manager.PermissionAccess) (bool, engine.AppError)
	Create(profile *model.CognitiveProfile) (*model.CognitiveProfile, engine.AppError)
	GetAllPage(domainId int64, req *model.SearchCognitiveProfile) ([]*model.CognitiveProfile, engine.AppError)
	GetAllPageByGroups(domainId int64, groups []int, search *model.SearchCognitiveProfile) ([]*model.CognitiveProfile, engine.AppError)
	Get(id, domainId int64) (*model.CognitiveProfile, engine.AppError)
	Update(profile *model.CognitiveProfile) (*model.CognitiveProfile, engine.AppError)
	Delete(domainId, id int64) engine.AppError
	GetById(id int64) (*model.CognitiveProfile, engine.AppError)

	SearchTtsProfile(domainId int64, profileId int) (*model.TtsProfile, engine.AppError)
}

type TranscriptFileStore interface {
	Store(t *model.FileTranscript) (*model.FileTranscript, engine.AppError)

	CreateJobs(domainId int64, params model.TranscriptOptions) ([]*model.FileTranscriptJob, engine.AppError)
	GetPhrases(domainId, id int64, search *model.ListRequest) ([]*model.TranscriptPhrase, engine.AppError)
	Delete(domainId int64, ids []int64, uuid []string) ([]int64, engine.AppError)
	Put(ctx context.Context, domainId int64, uuid string, tr model.FileTranscript) (int64, engine.AppError)
}

type ImportTemplateStore interface {
	CheckAccess(domainId int64, id int32, groups []int, access auth_manager.PermissionAccess) (bool, engine.AppError)
	Create(domainId int64, template *model.ImportTemplate) (*model.ImportTemplate, engine.AppError)
	GetAllPage(domainId int64, req *model.SearchImportTemplate) ([]*model.ImportTemplate, engine.AppError)
	GetAllPageByGroups(domainId int64, groups []int, search *model.SearchImportTemplate) ([]*model.ImportTemplate, engine.AppError)
	Get(domainId int64, id int32) (*model.ImportTemplate, engine.AppError)
	Update(domainId int64, template *model.ImportTemplate) (*model.ImportTemplate, engine.AppError)
	Delete(domainId int64, id int32) engine.AppError
}

type FilePoliciesStore interface {
	Create(ctx context.Context, domainId int64, policy *model.FilePolicy) (*model.FilePolicy, engine.AppError)
	GetAllPage(ctx context.Context, domainId int64, req *model.SearchFilePolicy) ([]*model.FilePolicy, engine.AppError)
	Get(ctx context.Context, domainId int64, id int32) (*model.FilePolicy, engine.AppError)
	Update(ctx context.Context, domainId int64, policy *model.FilePolicy) (*model.FilePolicy, engine.AppError)
	Delete(ctx context.Context, domainId int64, id int32) engine.AppError
	ChangePosition(ctx context.Context, domainId int64, fromId, toId int32) engine.AppError
	// AllByDomainId internal
	AllByDomainId(ctx context.Context, domainId int64) ([]model.FilePolicy, engine.AppError)
}

type SystemSettingsStore interface {
	ValueByName(ctx context.Context, domainId int64, name string) (engine.SysValue, engine.AppError)
}
