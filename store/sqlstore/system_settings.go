package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/store"
)

type SqlSysSettingsStore struct {
	SqlStore
}

func NewSqlSysSettingsStore(sqlStore SqlStore) store.SystemSettingsStore {
	us := &SqlSysSettingsStore{sqlStore}
	return us
}

func (s *SqlSysSettingsStore) ValueByName(ctx context.Context, domainId int64, name string) (engine.SysValue, engine.AppError) {
	var outValue engine.SysValue
	err := s.GetReplica().WithContext(ctx).SelectOne(&outValue, `select s.value
from call_center.system_settings s
where domain_id = :DomainId::int8 and name = :Name::varchar`, map[string]interface{}{
		"DomainId": domainId,
		"Name":     name,
	})

	if err != nil && err != sql.ErrNoRows {
		return nil, engine.NewCustomCodeError("store.sql_sys_settings.value.app_error", fmt.Sprintf("Name=%v, %s", name, err.Error()), extractCodeFromErr(err))
	}

	return outValue, nil
}
