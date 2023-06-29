package sqlstore

import (
	"database/sql"

	"github.com/go-gorp/gorp"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/store"
)

type SqlJobStore struct {
	SqlStore
}

func NewSqlJobStore(sqlStore SqlStore) store.JobStore {
	s := &SqlJobStore{sqlStore}

	return s
}

func (jss SqlJobStore) CreateIndexesIfNotExists() {

}

func (jss SqlJobStore) Save(job *model.Job) (*model.Job, engine.AppError) {

	err := jss.GetMaster().SelectOne(job, `insert into storage.jobs (id, type, priority, schedule_id, schedule_time, create_at, start_at, last_activity_at, status,
                  progress, data)
values (:Id, :Type, :Priority, :ScheduleId, :ScheduleTime, :CreatedAt, :StartAt, :LastActivityAt, :Status, :Progress, :Data)
returning *`, map[string]interface{}{
		"Id":             job.Id,
		"Type":           job.Type,
		"Priority":       job.Priority,
		"ScheduleId":     job.ScheduleId,
		"ScheduleTime":   job.ScheduleTime,
		"CreatedAt":      job.CreateAt,
		"StartAt":        job.StartAt,
		"LastActivityAt": job.LastActivityAt,
		"Status":         job.Status,
		"Progress":       job.Progress,
		"Data":           model.MapToJson(job.Data),
	})

	if err != nil {
		return nil, engine.NewInternalError("store.sql_job.save.app_error", "id="+job.Id+", "+err.Error())
	}

	return job, nil
}

func (jss SqlJobStore) UpdateOptimistically(job *model.Job, currentStatus string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		if sqlResult, err := jss.GetMaster().Exec(
			`UPDATE
				storage.jobs
			SET
				last_activity_at = :LastActivityAt,
				status = :Status,
				progress = :Progress,
				data = :Data
			WHERE
				id = :Id
			AND
				status = :OldStatus`,
			map[string]interface{}{
				"Id":             job.Id,
				"OldStatus":      currentStatus,
				"LastActivityAt": model.GetMillis(),
				"Status":         job.Status,
				"Data":           job.DataToJson(),
				"Progress":       job.Progress,
			}); err != nil {
			result.Err = engine.NewInternalError("store.sql_job.update.app_error", "id="+job.Id+", "+err.Error())
		} else {
			rows, err := sqlResult.RowsAffected()

			if err != nil {
				result.Err = engine.NewInternalError("store.sql_job.update.app_error", "id="+job.Id+", "+err.Error())
			} else {
				if rows == 1 {
					result.Data = true
				} else {
					result.Data = false
				}
			}
		}
	})
}

func (jss SqlJobStore) UpdateStatus(id string, status string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		job := &model.Job{
			Id:             id,
			Status:         status,
			LastActivityAt: model.GetMillis(),
		}

		if _, err := jss.GetMaster().UpdateColumns(func(col *gorp.ColumnMap) bool {
			return col.ColumnName == "status" || col.ColumnName == "last_activity_at"
		}, job); err != nil {
			result.Err = engine.NewInternalError("store.sql_job.update.app_error", "id="+id+", "+err.Error())
		}

		if result.Err == nil {
			result.Data = job
		}
	})
}

func (jss SqlJobStore) UpdateStatusOptimistically(id string, currentStatus string, newStatus string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var startAtClause string
		if newStatus == model.JOB_STATUS_IN_PROGRESS {
			startAtClause = `start_at = :StartAt,`
		}

		if sqlResult, err := jss.GetMaster().Exec(
			`UPDATE
				storage.jobs
			SET `+startAtClause+`
				status = :NewStatus,
				last_activity_at = :LastActivityAt
			WHERE
				id = :Id
			AND
				status = :OldStatus`, map[string]interface{}{"Id": id, "OldStatus": currentStatus, "NewStatus": newStatus, "StartAt": model.GetMillis(), "LastActivityAt": model.GetMillis()}); err != nil {
			result.Err = engine.NewInternalError("store.sql_job.update.app_error", "id="+id+", "+err.Error())
		} else {
			rows, err := sqlResult.RowsAffected()

			if err != nil {
				result.Err = engine.NewInternalError("store.sql_job.update.app_error", "id="+id+", "+err.Error())
			} else {
				if rows == 1 {
					result.Data = true
				} else {
					result.Data = false
				}
			}
		}
	})
}

func (jss SqlJobStore) Get(id string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var status *model.Job

		if err := jss.GetReplica().SelectOne(&status,
			`SELECT
				*
			FROM
				storage.jobs
			WHERE
				id = :Id`, map[string]interface{}{"Id": id}); err != nil {
			if err == sql.ErrNoRows {
				result.Err = engine.NewNotFoundError("store.sql_job.get.app_error", "Id="+id+", "+err.Error())
			} else {
				result.Err = engine.NewInternalError("store.sql_job.get.app_error", "Id="+id+", "+err.Error())
			}
		} else {
			result.Data = status
		}
	})
}

func (jss SqlJobStore) GetAllPage(offset int, limit int) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var statuses []*model.Job

		if _, err := jss.GetReplica().Select(&statuses,
			`SELECT
				*
			FROM
				storage.jobs
			ORDER BY
				create_at DESC
			LIMIT
				:Limit
			OFFSET
				:Offset`, map[string]interface{}{"Limit": limit, "Offset": offset}); err != nil {
			result.Err = engine.NewInternalError("store.sql_job.get_all.app_error", err.Error())
		} else {
			result.Data = statuses
		}
	})
}

func (jss SqlJobStore) GetAllByType(jobType string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var statuses []*model.Job

		if _, err := jss.GetReplica().Select(&statuses,
			`SELECT
				*
			FROM
				storage.jobs
			WHERE
				type = :Type
			ORDER BY
				create_at DESC`, map[string]interface{}{"Type": jobType}); err != nil {
			result.Err = engine.NewInternalError("store.sql_job.get_all.app_error", "Type="+jobType+", "+err.Error())
		} else {
			result.Data = statuses
		}
	})
}

func (jss SqlJobStore) GetAllByTypePage(jobType string, offset int, limit int) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var statuses []*model.Job

		if _, err := jss.GetReplica().Select(&statuses,
			`SELECT
				*
			FROM
				storage.jobs
			WHERE
				type = :Type
			ORDER BY
				create_at DESC
			LIMIT
				:Limit
			OFFSET
				:Offset`, map[string]interface{}{"Type": jobType, "Limit": limit, "Offset": offset}); err != nil {
			result.Err = engine.NewInternalError("store.sql_job.get_all.app_error", "Type="+jobType+", "+err.Error())
		} else {
			result.Data = statuses
		}
	})
}

func (jss SqlJobStore) GetAllByStatus(status string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var statuses []*model.Job

		if _, err := jss.GetReplica().Select(&statuses,
			`SELECT
				*
			FROM
				storage.jobs
			WHERE
				status = :Status
			ORDER BY
				create_at ASC`, map[string]interface{}{"Status": status}); err != nil {
			result.Err = engine.NewInternalError("store.sql_job.get_all.app_error", "Status="+status+", "+err.Error())
		} else {
			result.Data = statuses
		}
	})
}

func (jss SqlJobStore) GetNewestJobByStatusAndType(status string, jobType string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var job *model.Job

		if err := jss.GetReplica().SelectOne(&job,
			`SELECT
				*
			FROM
				storage.jobs
			WHERE
				status = :Status
			AND
				type = :Type
			ORDER BY
				create_at DESC
			LIMIT 1`, map[string]interface{}{"Status": status, "Type": jobType}); err != nil && err != sql.ErrNoRows {
			result.Err = engine.NewInternalError("store.sql_job.get_newest_job_by_status_and_type.app_error", "Status="+status+", "+err.Error())
		} else {
			result.Data = job
		}
	})
}

func (jss SqlJobStore) GetCountByStatusAndType(status string, jobType string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		if count, err := jss.GetReplica().SelectInt(`SELECT
				COUNT(*)
			FROM
				storage.jobs
			WHERE
				status = :Status
			AND
				type = :Type`, map[string]interface{}{"Status": status, "Type": jobType}); err != nil {
			result.Err = engine.NewInternalError("store.sql_job.get_count_by_status_and_type.app_error", "Status="+status+", "+err.Error())
		} else {
			result.Data = count
		}
	})
}

func (jss SqlJobStore) GetAllByStatusAndLessScheduleTime(status string, t int64) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var statuses []*model.Job

		if _, err := jss.GetReplica().Select(&statuses,
			`SELECT
				*
			FROM
				storage.jobs
			WHERE
				status = :Status AND schedule_time <= :Time
			ORDER BY
				create_at ASC`, map[string]interface{}{"Status": status, "Time": t}); err != nil {
			result.Err = engine.NewInternalError("store.sql_job.get_all.app_error", "Status="+status+", "+err.Error())
		} else {
			result.Data = statuses
		}
	})
}

func (jss SqlJobStore) Delete(id string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		if _, err := jss.GetMaster().Exec(
			`DELETE FROM
				storage.jobs
			WHERE
				id = :Id`, map[string]interface{}{"Id": id}); err != nil {
			result.Err = engine.NewInternalError("store.sql_job.delete.app_error", "id="+id+", "+err.Error())
		} else {
			result.Data = id
		}
	})
}
