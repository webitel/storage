package sqlstore

import (
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/store"
)

type SqlUploadJobStore struct {
	SqlStore
}

func NewSqlUploadJobStore(sqlStore SqlStore) store.UploadJobStore {
	us := &SqlUploadJobStore{sqlStore}
	return us
}

func (self *SqlUploadJobStore) CreateIndexesIfNotExists() {

}

func (self *SqlUploadJobStore) Create(job *model.JobUploadFile) (*model.JobUploadFile, engine.AppError) {
	job.PreSave()
	id, err := self.GetMaster().SelectInt(`insert into storage.upload_file_jobs (name, uuid, mime_type, size, instance,
                                      created_at, updated_at, domain_id, view_name)
values (:Name, :Uuid, :Mime, :Size, :Instance, :CreatedAt, :UpdatedAt, :DomainId, :VName)
returning id
`, map[string]interface{}{
		"Name":      job.Name,
		"Uuid":      job.Uuid,
		"Mime":      job.MimeType,
		"Size":      job.Size,
		"Instance":  job.Instance,
		"CreatedAt": job.CreatedAt,
		"UpdatedAt": job.UpdatedAt,
		"DomainId":  job.DomainId,
		"VName":     job.ViewName,
	})

	if err != nil {
		return nil, engine.NewInternalError("store.sql_upload_job.save.app_error", err.Error())
	}

	job.Id = id
	return job, nil
}

func (self *SqlUploadJobStore) GetAllPageByInstance(limit int, instance string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var jobs []*model.JobUploadFile

		res, err := self.GetReplica().Query("SELECT id, name, uuid, domain_id, mime_type, size, email_msg, email_sub, instance, attempts "+
			"		FROM storage.upload_file_jobs LIMIT $1", limit)
		if err != nil {
			result.Err = engine.NewInternalError("store.sql_upload_job.list.app_error", err.Error())
			return
		}
		defer res.Close()

		for res.Next() {
			job := new(model.JobUploadFile)
			err = res.Scan(&job.Id, &job.Name, &job.Uuid, &job.DomainId, &job.MimeType, &job.Size, &job.EmailMsg, &job.EmailSub, &job.Instance, &job.Attempts)
			if err != nil {
				result.Err = engine.NewInternalError("store.sql_upload_job.list.app_error", err.Error())
				return
			}
			jobs = append(jobs, job)
		}
		result.Data = jobs
	})
}

func (self *SqlUploadJobStore) UpdateWithProfile(limit int, instance string, betweenAttemptSec int64, defStore bool) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var jobs []*model.JobUploadFileWithProfile

		_, err := self.GetMaster().Select(&jobs, `update storage.upload_file_jobs uu
set attempts = attempts + 1
  ,state = 1
  ,updated_at = extract(EPOCH from now()) :: BIGINT
from (
       SELECT
         t.id,
         t.name,
         t.uuid,

         t.domain_id,
         t.mime_type,
         t.size,
         t.email_msg,
         t.email_sub,
         profile.id as profile_id,
         profile.updated_at profile_updated_at
       FROM storage.upload_file_jobs as t
         left join lateral (              select
                                             tmp.domain_id,
                                             tmp.id,
                                             tmp.updated_at
                                           from (select
                                                   p1.domain_id,
                                                   p1.id,
                                                   p1.updated_at,
                                                   p1.priority
                                                 from storage.file_backend_profiles p1
                                                 where p1.domain_id = t.domain_id and NOT p1.disabled is TRUE) as tmp
                                           order by tmp.priority desc
                                           FETCH FIRST 1 ROW ONLY              ) profile ON profile.domain_id = t.domain_id
       WHERE state = 0  and (:UseDef::bool = true or profile.id notnull ) AND instance = :Instance AND (t.updated_at < :UpdatedAt OR attempts = 0)
       ORDER BY created_at ASC
       LIMIT :Limit) tmp
WHERE tmp.id = uu.id and state = 0
returning tmp.*`, map[string]interface{}{
			"UseDef":    defStore,
			"Instance":  instance,
			"Limit":     limit,
			"UpdatedAt": model.GetMillis() - betweenAttemptSec,
		})
		if err != nil {
			result.Err = engine.NewInternalError("store.sql_upload_job.update_with_profile.app_error", err.Error())
			return
		}

		result.Data = jobs
	})
}

func (self *SqlUploadJobStore) SetStateError(id int, errMsg string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		self.GetMaster().Exec(`update storage.upload_file_jobs
set state = 0,
  updated_at = $2
where id = $1`, id, model.GetMillis())
	})
}

func (self *SqlUploadJobStore) RemoveById(id int) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		panic("TODO")
	})
}
