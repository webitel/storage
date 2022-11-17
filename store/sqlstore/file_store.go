package sqlstore

import (
	"fmt"
	"net/http"

	"github.com/lib/pq"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/store"
)

type SqlFileStore struct {
	SqlStore
}

func NewSqlFileStore(sqlStore SqlStore) store.FileStore {
	us := &SqlFileStore{sqlStore}

	return us
}

func (self SqlFileStore) CreateIndexesIfNotExists() {

}

func (self SqlFileStore) Create(file *model.File) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		id, err := self.GetMaster().SelectInt(`
			insert into storage.files(id, name, uuid, profile_id, size, domain_id, mime_type, properties, created_at, instance, view_name)
            values(nextval('storage.upload_file_jobs_id_seq'::regclass), :Name, :Uuid, null, :Size, :DomainId, :Mime, :Props, :CreatedAt, :Inst, :VName)
			returning id
		`, map[string]interface{}{
			"Name":      file.Name,
			"Uuid":      file.Uuid,
			"Size":      file.Size,
			"DomainId":  file.DomainId,
			"Mime":      file.MimeType,
			"Props":     file.Properties.ToJson(),
			"CreatedAt": file.CreatedAt,
			"Inst":      file.Instance,
			"VName":     file.ViewName,
		})

		if err != nil {
			result.Err = model.NewAppError("SqlFileStore.Create", "store.sql_file.create.app_error", nil, err.Error(), http.StatusInternalServerError)
		} else {
			result.Data = id
		}
	})
}

func (self SqlFileStore) MarkRemove(domainId int64, ids []int64) *model.AppError {
	_, err := self.GetMaster().Exec(`update storage.files
set removed = true
where domain_id = :DomainId and id = any(:Ids::int8[])`, map[string]interface{}{
		"DomainId": domainId,
		"Ids":      pq.Array(ids),
	})

	if err != nil {
		return model.NewAppError("SqlFileStore.MarkRemove", "store.sql_file.remove.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return nil
}

//TODO reference tables ?
func (self SqlFileStore) MoveFromJob(jobId int64, profileId *int, properties model.StringInterface) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		_, err := self.GetMaster().Exec(`with del as (
  delete from storage.upload_file_jobs
  where id = $1
  returning id, name, uuid, size, domain_id, mime_type, created_at, instance, view_name
)
insert into storage.files(id, name, uuid, profile_id, size, domain_id, mime_type, properties, created_at, instance, view_name)
select del.id, del.name, del.uuid, $2, del.size, del.domain_id, del.mime_type, $3, del.created_at, del.instance, del.view_name
from del`, jobId, profileId, properties.ToJson())

		if err != nil {
			result.Err = model.NewAppError("SqlFileStore.MoveFromJob", "store.sql_file.move_from_job.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	})
}

func (self SqlFileStore) GetAllPageByDomain(domain string, offset, limit int) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var recordings []*model.File

		query := `SELECT * FROM files 
			WHERE domain = :Domain  
			LIMIT :Limit OFFSET :Offset`

		if _, err := self.GetReplica().Select(&recordings, query, map[string]interface{}{"Domain": domain, "Offset": offset, "Limit": limit}); err != nil {
			result.Err = model.NewAppError("SqlFileStore.List", "store.sql_file.get_all.finding.app_error", nil, err.Error(), http.StatusInternalServerError)
		} else {
			result.Data = recordings
		}
	})
}

func (s SqlFileStore) GetFileWithProfile(domainId, id int64) (*model.FileWithProfile, *model.AppError) {
	var file *model.FileWithProfile
	err := s.GetReplica().SelectOne(&file, `SELECT f.*, p.updated_at as profile_updated_at
	FROM storage.files f
		left join storage.file_backend_profiles p on p.id = f.profile_id
	WHERE f.id = :Id
	  AND f.domain_id = :DomainId`, map[string]interface{}{
		"Id":       id,
		"DomainId": domainId,
	})

	if err != nil {
		return nil, model.NewAppError("SqlFileStore.GetFileWithProfile", "store.sql_file.get_with_profile.app_error", nil,
			fmt.Sprintf("Id=%d %s", id, err.Error()), extractCodeFromErr(err))
	}
	return file, nil
}

func (s SqlFileStore) GetFileByUuidWithProfile(domainId int64, uuid string) (*model.FileWithProfile, *model.AppError) {
	var file *model.FileWithProfile
	err := s.GetReplica().SelectOne(&file, `SELECT f.*, p.updated_at as profile_updated_at
	FROM storage.files f
		left join storage.file_backend_profiles p on p.id = f.profile_id
	WHERE f.uuid = :Uuid
	  AND f.domain_id = :DomainId
	order by created_at desc
	limit 1`, map[string]interface{}{
		"Uuid":     uuid,
		"DomainId": domainId,
	})

	if err != nil {
		return nil, model.NewAppError("SqlFileStore.GetFileByUuidWithProfile", "store.sql_file.get_by_uuid_with_profile.app_error", nil,
			fmt.Sprintf("Uuid=%d %s", uuid, err.Error()), extractCodeFromErr(err))
	}
	return file, nil
}
