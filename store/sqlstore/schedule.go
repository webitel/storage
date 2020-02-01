package sqlstore

import (
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/store"
	"net/http"
)

type SqlScheduleStore struct {
	SqlStore
}

func NewSqlScheduleStore(sqlStore SqlStore) store.ScheduleStore {
	us := &SqlScheduleStore{sqlStore}
	return us
}

func (self SqlScheduleStore) CreateIndexesIfNotExists() {

}

func (self SqlScheduleStore) GetAllEnablePage(limit, offset int) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var data []*model.Schedule

		query := `SELECT * FROM schedulers
			WHERE enabled is TRUE
			LIMIT :Limit OFFSET :Offset`

		if _, err := self.GetReplica().Select(&data, query, map[string]interface{}{"Offset": offset, "Limit": limit}); err != nil {
			result.Err = model.NewAppError("SqlScheduleStore.List", "store.sql_schedule.get_all.finding.app_error", nil, err.Error(), http.StatusInternalServerError)
		} else {
			result.Data = data
		}
	})
}

func (self SqlScheduleStore) GetAllPageByType(typeName string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var data []*model.Schedule

		query := `SELECT * FROM schedulers
			WHERE enabled is TRUE AND type = :Type`

		if _, err := self.GetReplica().Select(&data, query, map[string]interface{}{"Type": typeName}); err != nil {
			result.Err = model.NewAppError("SqlScheduleStore.GetAllPageByType", "store.sql_schedule.get_all_by_type.finding.app_error", nil, err.Error(), http.StatusInternalServerError)
		} else {
			result.Data = data
		}
	})
}

func (self SqlScheduleStore) GetAllWithNoJobs(limit, offset int) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var data []*model.Schedule

		query := `SELECT * FROM schedulers
			WHERE enabled is TRUE AND NOT EXISTS (SELECT null FROM jobs WHERE jobs.status = :JobStatus AND jobs.type = schedulers.type AND jobs.schedule_id = schedulers.id)
			LIMIT :Limit OFFSET :Offset`

		if _, err := self.GetReplica().Select(&data, query, map[string]interface{}{"Offset": offset, "Limit": limit, "JobStatus": model.JOB_STATUS_PENDING}); err != nil {
			result.Err = model.NewAppError("SqlScheduleStore.List", "store.sql_schedule.get_all.finding.app_error", nil, err.Error(), http.StatusInternalServerError)
		} else {
			result.Data = data
		}
	})
}
