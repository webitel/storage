package store

import (
	"time"

	"github.com/webitel/engine/auth_manager"

	"github.com/webitel/storage/model"
)

type StoreResult struct {
	Data interface{}
	Err  *model.AppError
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
}

type UploadJobStore interface {
	Create(job *model.JobUploadFile) (*model.JobUploadFile, *model.AppError)
	//Save(job *model.JobUploadFile) StoreChannel
	GetAllPageByInstance(limit int, instance string) StoreChannel
	UpdateWithProfile(limit int, instance string, betweenAttemptSec int64, defStore bool) StoreChannel
	SetStateError(id int, errMsg string) StoreChannel
}

type SyncFileStore interface {
	FetchJobs(limit int) ([]*model.SyncJob, *model.AppError)
	SetRemoveJobs(localExpDay int) *model.AppError
	Clean(jobId int64) *model.AppError
	Remove(jobId int64) *model.AppError

	RemoveErrors() *model.AppError
	SetError(jobId int64, e error) *model.AppError
}

type FileBackendProfileStore interface {
	CheckAccess(domainId, id int64, groups []int, access auth_manager.PermissionAccess) (bool, *model.AppError)
	Create(profile *model.FileBackendProfile) (*model.FileBackendProfile, *model.AppError)
	GetAllPage(domainId int64, req *model.SearchFileBackendProfile) ([]*model.FileBackendProfile, *model.AppError)
	GetAllPageByGroups(domainId int64, groups []int, search *model.SearchFileBackendProfile) ([]*model.FileBackendProfile, *model.AppError)
	Get(id, domainId int64) (*model.FileBackendProfile, *model.AppError)
	GetById(id int) (*model.FileBackendProfile, *model.AppError)
	Update(profile *model.FileBackendProfile) (*model.FileBackendProfile, *model.AppError)
	Delete(domainId, id int64) *model.AppError

	GetAllPageByDomain(domain string, limit, offset int) StoreChannel
}

type FileStore interface {
	Create(file *model.File) StoreChannel
	GetFileWithProfile(domainId, id int64) (*model.FileWithProfile, *model.AppError)
	GetFileByUuidWithProfile(domainId int64, uuid string) (*model.FileWithProfile, *model.AppError)
	MarkRemove(domainId int64, ids []int64) *model.AppError

	GetAllPageByDomain(domain string, offset, limit int) StoreChannel
	MoveFromJob(jobId int64, profileId *int, properties model.StringInterface) StoreChannel
}

type MediaFileStore interface {
	Create(file *model.MediaFile) (*model.MediaFile, *model.AppError)
	GetAllPage(domainId int64, search *model.SearchMediaFile) ([]*model.MediaFile, *model.AppError)
	Get(domainId int64, id int) (*model.MediaFile, *model.AppError)
	Delete(domainId, id int64) *model.AppError

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
	Save(job *model.Job) (*model.Job, *model.AppError)
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
	CheckAccess(domainId, id int64, groups []int, access auth_manager.PermissionAccess) (bool, *model.AppError)
	Create(profile *model.CognitiveProfile) (*model.CognitiveProfile, *model.AppError)
	GetAllPage(domainId int64, req *model.SearchCognitiveProfile) ([]*model.CognitiveProfile, *model.AppError)
	GetAllPageByGroups(domainId int64, groups []int, search *model.SearchCognitiveProfile) ([]*model.CognitiveProfile, *model.AppError)
	Get(id, domainId int64) (*model.CognitiveProfile, *model.AppError)
	Update(profile *model.CognitiveProfile) (*model.CognitiveProfile, *model.AppError)
	Delete(domainId, id int64) *model.AppError
	GetById(id int64) (*model.CognitiveProfile, *model.AppError)

	SearchTtsProfile(domainId int64, profileId int) (*model.TtsProfile, *model.AppError)
}

type TranscriptFileStore interface {
	GetByFileId(fileId int64, profileId int64) (*model.FileTranscript, *model.AppError)
	Store(t *model.FileTranscript) (*model.FileTranscript, *model.AppError)

	CreateJobs(domainId int64, params model.TranscriptOptions) ([]*model.FileTranscriptJob, *model.AppError)
	GetPhrases(domainId, id int64, search *model.ListRequest) ([]*model.TranscriptPhrase, *model.AppError)
	Delete(domainId int64, ids []int64, uuid []string) ([]int64, *model.AppError)
}

type ImportTemplateStore interface {
	CheckAccess(domainId int64, id int32, groups []int, access auth_manager.PermissionAccess) (bool, *model.AppError)
	Create(domainId int64, template *model.ImportTemplate) (*model.ImportTemplate, *model.AppError)
	GetAllPage(domainId int64, req *model.SearchImportTemplate) ([]*model.ImportTemplate, *model.AppError)
	GetAllPageByGroups(domainId int64, groups []int, search *model.SearchImportTemplate) ([]*model.ImportTemplate, *model.AppError)
	Get(domainId int64, id int32) (*model.ImportTemplate, *model.AppError)
	Update(domainId int64, template *model.ImportTemplate) (*model.ImportTemplate, *model.AppError)
	Delete(domainId int64, id int32) *model.AppError
}
