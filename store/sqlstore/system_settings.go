package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/store"
)

type SqlSysSettingsStore struct {
	SqlStore
}

func NewSqlSysSettingsStore(sqlStore SqlStore) store.SystemSettingsStore {
	us := &SqlSysSettingsStore{sqlStore}
	return us
}

func (s *SqlSysSettingsStore) ValueByName(ctx context.Context, domainId int64, name string) (model.SysValue, model.AppError) {
	var outValue model.SysValue
	err := s.GetReplica().WithContext(ctx).SelectOne(&outValue, `select s.value
from call_center.system_settings s
where domain_id = :DomainId::int8 and name = :Name::varchar`, map[string]interface{}{
		"DomainId": domainId,
		"Name":     name,
	})

	if err != nil && err != sql.ErrNoRows {
		return nil, model.NewCustomCodeError("store.sql_sys_settings.value.app_error", fmt.Sprintf("Name=%v, %s", name, err.Error()), extractCodeFromErr(err))
	}

	return outValue, nil
}
