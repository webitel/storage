package sqlstore

import (
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/store"
)

type SqlSyncFileStore struct {
	SqlStore
}

func NewSqlSyncFileStore(sqlStore SqlStore) store.SyncFileStore {
	us := &SqlSyncFileStore{sqlStore}
	return us
}

func (s SqlSyncFileStore) FetchJobs(limit int) ([]*model.SyncJob, engine.AppError) {
	var res []*model.SyncJob
	_, err := s.GetMaster().Select(&res, `update storage.file_jobs u
set state = 1
from (
    select j.id, j.file_id, f.domain_id, f.properties, f.profile_id, p.updated_at as profile_updated_at, f.name, f.size, f.mime_type, f.instance,
		j.action, j.config
    from storage.file_jobs j
        inner join storage.files f on f.id = j.file_id
        left join storage.file_backend_profiles p on p.id = f.profile_id
    where j.state = 0
    order by j.created_at asc
    limit :Limit
    for update OF j skip locked
) j
where u.id = j.id and u.state = 0
returning j.*`, map[string]interface{}{
		"Limit": limit,
	})

	if err != nil {
		return nil, engine.NewInternalError("store.sql_sync_file_job.save.app_error", err.Error())
	}

	return res, nil
}

func (s SqlSyncFileStore) SetRemoveJobs(localExpDay int) engine.AppError {
	_, err := s.GetMaster().Exec(`insert into storage.file_jobs (file_id, action)
select f.id, :Action
from (
    select id
    from storage.files
    where retention_until < now()
    order by retention_until
    limit 1000
 ) f
union all
select id, :Action
from (
    select f.id
    from storage.files f
    where f.removed
        and not exists(select 1 from storage.file_jobs j where j.file_id = f.id)
    order by f.created_at
	limit 1000
) t;`, map[string]interface{}{
		"LocalExpire": localExpDay,
		"Action":      model.SyncJobRemove,
	})

	if err != nil {
		return engine.NewInternalError("store.sql_sync_file_job.set_removed.app_error", err.Error())
	}

	return nil
}

func (s SqlSyncFileStore) Clean(jobId int64) engine.AppError {
	_, err := s.GetMaster().Exec(`with del as (
    delete
    from storage.file_jobs rj
    where rj.id = :Id
    returning rj.file_id
)
delete
from storage.files f
where f.id = (select del.file_id from del )`, map[string]interface{}{
		"Id": jobId,
	})

	if err != nil {
		return engine.NewInternalError("store.sql_sync_file_job.clean.app_error", err.Error())
	}

	return nil
}

func (s SqlSyncFileStore) Remove(jobId int64) engine.AppError {
	_, err := s.GetMaster().Exec(`    delete
    from storage.file_jobs rj
    where rj.id = :Id`, map[string]interface{}{
		"Id": jobId,
	})

	if err != nil {
		return engine.NewCustomCodeError("store.sql_sync_file_job.remove.app_error", err.Error(), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlSyncFileStore) RemoveErrors() engine.AppError {
	_, err := s.GetMaster().Exec(`delete
from storage.file_jobs j
where j.updated_at < now() - interval '1h' and j.state = 3`)
	if err != nil {
		return engine.NewCustomCodeError("store.sql_sync_file_job.remove_err.app_error", err.Error(), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlSyncFileStore) SetError(jobId int64, e error) engine.AppError {
	_, err := s.GetMaster().Exec(`update storage.file_jobs
	set error = :Error ,
		state = 3,
		updated_at = now()
	where id = :Id`, map[string]interface{}{
		"Error": e.Error(),
		"Id":    jobId,
	})

	if err != nil {
		return engine.NewCustomCodeError("store.sql_sync_file_job.set_err.app_error", err.Error(), extractCodeFromErr(err))
	}

	return nil
}
