package sqlstore

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/webitel/engine/pkg/wbt/auth_manager"
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

func (self *SqlFileStore) CreateIndexesIfNotExists() {

}

func (self *SqlFileStore) GetAllPage(ctx context.Context, domainId int64, search *model.SearchFile) ([]*model.File, model.AppError) {
	var files []*model.File

	f := map[string]interface{}{
		"DomainId":     domainId,
		"Ids":          pq.Array(search.Ids),
		"ReferenceIds": pq.Array(search.ReferenceIds),
	}

	err := self.ListQueryCtx(ctx, &files, search.ListRequest,
		`domain_id = :DomainId
				and (:Ids::int[] isnull or id = any(:Ids))
				and (:ReferenceIds::varchar[] isnull or reference_id = any(:ReferenceIds))
		`,
		model.File{}, f)

	if err != nil {
		return nil, model.NewCustomCodeError("store.sql_file.get_all.finding.app_error", err.Error(), extractCodeFromErr(err))
	}

	return files, nil
}

func (self SqlFileStore) Create(file *model.File) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		id, err := self.GetMaster().SelectInt(`
			insert into storage.files(id, name, uuid, size, domain_id, mime_type, properties, created_at, instance, view_name, 
			                          profile_id, sha256sum, channel, thumbnail, retention_until, uploaded_by)
            values(nextval('storage.upload_file_jobs_id_seq'::regclass), :Name, :Uuid, :Size, :DomainId, :Mime, :Props, :CreatedAt, :Inst, :VName, 
                   :ProfileId, :SHA256Sum, :Channel, :Thumbnail::jsonb, :RetentionUntil::timestamptz, :UploadedBy::int8)
			returning id
		`, map[string]interface{}{
			"Name":           file.Name,
			"Uuid":           file.Uuid,
			"Size":           file.Size,
			"DomainId":       file.DomainId,
			"Mime":           file.MimeType,
			"Props":          file.Properties.ToJson(),
			"CreatedAt":      file.CreatedAt,
			"Inst":           file.Instance,
			"VName":          file.ViewName,
			"ProfileId":      file.ProfileId,
			"SHA256Sum":      file.SHA256Sum,
			"Channel":        file.Channel,
			"Thumbnail":      file.Thumbnail.ToJson(),
			"RetentionUntil": file.RetentionUntil,
			"UploadedBy":     file.UploadedBy.GetSafeId(),
		})

		if err != nil {
			result.Err = model.NewInternalError("store.sql_file.create.app_error", err.Error())
		} else {
			result.Data = id
		}
	})
}

func (self SqlFileStore) MarkRemove(domainId int64, ids []int64) model.AppError {
	_, err := self.GetMaster().Exec(`update storage.files
set removed = true
where domain_id = :DomainId and id = any(:Ids::int8[])`, map[string]interface{}{
		"DomainId": domainId,
		"Ids":      pq.Array(ids),
	})

	if err != nil {
		return model.NewCustomCodeError("store.sql_file.remove.app_error", err.Error(), extractCodeFromErr(err))
	}

	return nil
}

// TODO reference tables ?
func (self SqlFileStore) MoveFromJob(jobId int64, profileId *int, properties model.StringInterface, retentionUntil *time.Time) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		_, err := self.GetMaster().Exec(`with del as (
  delete from storage.upload_file_jobs
  where id = $1
  returning id, name, uuid, size, domain_id, mime_type, created_at, instance, view_name, channel
)
insert into storage.files(id, name, uuid, profile_id, size, domain_id, mime_type, properties, created_at, instance, view_name, 
	channel, retention_until)
select del.id, del.name, del.uuid, $2, del.size, del.domain_id, del.mime_type, $3, del.created_at, del.instance, del.view_name, 
	del.channel, $4::timestamptz
from del`, jobId, profileId, properties.ToJson(), retentionUntil)

		if err != nil {
			result.Err = model.NewInternalError("store.sql_file.move_from_job.app_error", err.Error())
		}
	})
}

// get permissions of the call record for user
func (self SqlFileStore) CheckCallRecordPermissions(ctx context.Context, fileId int, userId int64, domainId int64, groups []int) (bool, model.AppError) {

	exists, err := self.GetReplica().WithContext(ctx).SelectInt(`
		select exists(select 1
		from call_center.cc_calls_history t
		where t.id = (select uuid
					  from storage.files f
					  where id = :FileId
					  limit 1)::uuid
		  and (
			(t.user_id = any (call_center.cc_calls_rbac_users(:Domain::int8, :CurrentUserId::int8) || :Groups::int[])
				or t.queue_id = any (call_center.cc_calls_rbac_queues(:Domain::int8, :CurrentUserId::int8, :Groups::int[]))
				or (t.user_ids notnull and t.user_ids::int[] &&
										   call_center.rbac_users_from_group(:Class::varchar, :Domain::int8, :Access::int2,
																					  :Groups::int[]))
				or (t.grantee_id = any (:Groups::int[]))
				)
			)
		)::int`, map[string]interface{}{
		"CurrentUserId": userId,
		"FileId":        fileId,
		"Domain":        domainId,
		"Groups":        pq.Array(groups),
		"Access":        auth_manager.PERMISSION_ACCESS_READ.Value(),
		"Class":         model.PERMISSION_SCOPE_RECORD_FILE,
	})
	if err != nil {
		return false, model.NewInternalError("store.sql_file.check_call_record_permissions.app_error", err.Error())
	}
	return exists == 1, nil

}

func (s SqlFileStore) GetFileWithProfile(domainId, id int64) (*model.FileWithProfile, model.AppError) {
	var file *model.FileWithProfile
	err := s.GetReplica().SelectOne(&file, `SELECT f.id,
       f.name,
       f.size,
       f.mime_type,
       f.properties,
       f.instance,
       f.uuid,
       f.profile_id,
       f.created_at,
       f.domain_id,
       f.view_name,
       f.channel,
       f.thumbnail,
       p.updated_at as profile_updated_at
FROM storage.files f
         left join storage.file_backend_profiles p on p.id = f.profile_id
	WHERE f.id = :Id
	  AND f.domain_id = :DomainId`, map[string]interface{}{
		"Id":       id,
		"DomainId": domainId,
	})

	if err != nil {
		return nil, model.NewCustomCodeError("store.sql_file.get_with_profile.app_error", fmt.Sprintf("Id=%d %s", id, err.Error()), extractCodeFromErr(err))
	}
	return file, nil
}

func (s SqlFileStore) GetFileByUuidWithProfile(domainId int64, uuid string) (*model.FileWithProfile, model.AppError) {
	var file *model.FileWithProfile
	err := s.GetReplica().SelectOne(&file, `SELECT f.id,
       f.name,
       f.size,
       f.mime_type,
       f.properties,
       f.instance,
       f.uuid,
       f.profile_id,
       f.created_at,
       f.domain_id,
       f.view_name,
       f.channel,
       f.thumbnail,
       p.updated_at as profile_updated_at
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
		return nil, model.NewCustomCodeError("store.sql_file.get_by_uuid_with_profile.app_error", fmt.Sprintf("Uuid=%d %s", uuid, err.Error()), extractCodeFromErr(err))
	}
	return file, nil
}

func (s SqlFileStore) Metadata(domainId int64, id int64) (model.BaseFile, model.AppError) {
	var m model.BaseFile
	err := s.GetReplica().SelectOne(&m, `select mime_type, coalesce(view_name, name) as name, size
from storage.files
where domain_id = :DomainId and id = :Id`, map[string]any{
		"DomainId": domainId,
		"Id":       id,
	})

	if err != nil {
		return model.BaseFile{}, model.NewCustomCodeError("store.sql_file.metadata.app_error", err.Error(), extractCodeFromErr(err))
	}

	return m, nil
}
