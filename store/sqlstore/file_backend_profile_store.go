package sqlstore

import (
	"fmt"

	"github.com/lib/pq"
	"github.com/webitel/engine/pkg/wbt/auth_manager"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/store"
)

type SqlFileBackendProfileStore struct {
	SqlStore
}

func NewSqlFileBackendProfileStore(sqlStore SqlStore) store.FileBackendProfileStore {
	us := &SqlFileBackendProfileStore{sqlStore}

	return us
}

func (self SqlFileBackendProfileStore) CreateIndexesIfNotExists() {

}

func (s SqlFileBackendProfileStore) CheckAccess(domainId, id int64, groups []int, access auth_manager.PermissionAccess) (bool, model.AppError) {

	res, err := s.GetReplica().SelectNullInt(`select 1
		where exists(
          select 1
          from file_backend_profiles_acl a
          where a.dc = :DomainId
            and a.object = :Id
            and a.subject = any (:Groups::int[])
            and a.access & :Access = :Access
        )`, map[string]interface{}{"DomainId": domainId, "Id": id, "Groups": pq.Array(groups), "Access": access.Value()})

	if err != nil {
		return false, nil
	}

	return res.Valid && res.Int64 == 1, nil
}

func (s SqlFileBackendProfileStore) Create(profile *model.FileBackendProfile) (*model.FileBackendProfile, model.AppError) {
	err := s.GetMaster().SelectOne(&profile, `with p as (
    insert into storage.file_backend_profiles (name, expire_day, priority, disabled, max_size_mb, properties, type,
                                           created_at, updated_at, created_by, updated_by,
                                           domain_id, description)
    values (:Name, :ExpireDay, :Priority, :Disabled, :MaxSize, :Properties, :Type, :CreatedAt, :UpdatedAt, :CreatedBy, :UpdatedBy,
            :DomainId, :Description)
    returning *
)
select p.id, call_center.cc_get_lookup(c.id, c.name) as created_by, p.created_at, call_center.cc_get_lookup(u.id, u.name) as updated_by,
       p.updated_at, p.name, p.description, p.expire_day as expire_days, p.priority, p.disabled, p.max_size_mb as max_size, p.properties,
       p.type, p.data_size, p.data_count
from p
    left join directory.wbt_user c on c.id = p.created_by
    left join directory.wbt_user u on u.id = p.updated_by`, map[string]interface{}{
		"Name":        profile.Name,
		"ExpireDay":   profile.ExpireDay,
		"Priority":    profile.Priority,
		"Disabled":    profile.Disabled,
		"MaxSize":     profile.MaxSizeMb,
		"Properties":  model.StringInterfaceToJson(profile.Properties),
		"Type":        profile.Type,
		"CreatedAt":   profile.CreatedAt,
		"UpdatedAt":   profile.UpdatedAt,
		"CreatedBy":   profile.CreatedBy.GetSafeId(),
		"UpdatedBy":   profile.UpdatedBy.GetSafeId(),
		"DomainId":    profile.DomainId,
		"Description": profile.Description,
	})

	if err != nil {
		return nil, model.NewCustomCodeError("store.sql_file_backend_profile.get.create.app_error", fmt.Sprintf("name=%v, %v", profile.Name, err.Error()), extractCodeFromErr(err))
	}

	return profile, nil
}

func (s SqlFileBackendProfileStore) GetAllPage(domainId int64, search *model.SearchFileBackendProfile) ([]*model.FileBackendProfile, model.AppError) {
	var profiles []*model.FileBackendProfile

	f := map[string]interface{}{
		"DomainId": domainId,
		"Ids":      pq.Array(search.Ids),
		"Q":        search.GetQ(),
	}

	err := s.ListQuery(&profiles, search.ListRequest,
		`domain_id = :DomainId
				and (:Ids::int[] isnull or id = any(:Ids))
				and (:Q::varchar isnull or (name ilike :Q::varchar ))`,
		model.FileBackendProfile{}, f)

	if err != nil {
		return nil, model.NewCustomCodeError("store.sql_file_backend_profile.get_all.finding.app_error", err.Error(), extractCodeFromErr(err))
	}

	return profiles, nil
}

func (s SqlFileBackendProfileStore) GetAllPageByGroups(domainId int64, groups []int, search *model.SearchFileBackendProfile) ([]*model.FileBackendProfile, model.AppError) {
	var profiles []*model.FileBackendProfile

	f := map[string]interface{}{
		"DomainId": domainId,
		"Ids":      pq.Array(search.Ids),
		"Q":        search.GetQ(),
		"Groups":   pq.Array(groups),
		"Access":   auth_manager.PERMISSION_ACCESS_READ.Value(),
	}

	err := s.ListQuery(&profiles, search.ListRequest,
		`domain_id = :DomainId
				and exists(select 1
				  from storage.file_backend_profiles_acl a
				  where a.dc = t.domain_id and a.object = t.id and a.subject = any(:Groups::int[]) and a.access&:Access = :Access)
				and (:Ids::int[] isnull or id = any(:Ids))
				and (:Q::varchar isnull or (name ilike :Q::varchar ))`,
		model.FileBackendProfile{}, f)

	if err != nil {
		return nil, model.NewCustomCodeError("store.sql_file_backend_profile.get_all.finding.app_error", err.Error(), extractCodeFromErr(err))
	}

	return profiles, nil
}

// FIXME
func (s SqlFileBackendProfileStore) Get(id, domainId int64) (*model.FileBackendProfile, model.AppError) {
	var profile *model.FileBackendProfile
	err := s.GetMaster().SelectOne(&profile, `select p.id, call_center.cc_get_lookup(c.id, c.name) as created_by, p.created_at, call_center.cc_get_lookup(u.id, u.name) as updated_by,
       p.updated_at, p.name, p.description, p.expire_day as expire_days, p.priority, p.disabled, p.max_size_mb as max_size, p.properties,
       p.type, coalesce(s.size, 0) data_size, coalesce(s.cnt, 0) data_count, p.domain_id
from storage.file_backend_profiles p
    left join lateral (
        select sum(s.size) size, sum(s.count) as cnt
        from storage.files_statistics s
        where s.profile_id = p.id
    ) s on true
    left join directory.wbt_user c on c.id = p.created_by
    left join directory.wbt_user u on u.id = p.updated_by
    where p.id = :Id and p.domain_id = :DomainId`, map[string]interface{}{
		"Id":       id,
		"DomainId": domainId,
	})

	if err != nil {
		return nil, model.NewCustomCodeError("store.sql_file_backend_profile.get.app_error", fmt.Sprintf("id=%d, domain=%d, %s", id, domainId, err.Error()), extractCodeFromErr(err))
	}

	return profile, nil
}

func (s SqlFileBackendProfileStore) Update(profile *model.FileBackendProfile) (*model.FileBackendProfile, model.AppError) {
	err := s.GetMaster().SelectOne(&profile, `with p as (
    update storage.file_backend_profiles
    set name = :Name,
        expire_day = :ExpireDay,
        priority = :Priority,
        disabled = :Disabled,
        max_size_mb = :MaxSize,
        properties = :Properties,
        description = :Description,
        updated_at = :UpdatedAt,
        updated_by = :UpdatedBy
    where id = :Id and domain_id = :DomainId
	returning *
)
select p.id, call_center.cc_get_lookup(c.id, c.name) as created_by, p.created_at, call_center.cc_get_lookup(u.id, u.name) as updated_by,
       p.updated_at, p.name, p.description, p.expire_day as expire_days, p.priority, p.disabled, p.max_size_mb as max_size, p.properties,
       p.type, coalesce(s.size, 0) data_size, coalesce(s.cnt, 0) data_count
from p
    left join lateral (
        select sum(s.size) size, sum(s.count) as cnt
        from storage.files_statistics s
        where s.profile_id = p.id
    ) s on true
    left join directory.wbt_user c on c.id = p.created_by
    left join directory.wbt_user u on u.id = p.updated_by`, map[string]interface{}{
		"Name":        profile.Name,
		"ExpireDay":   profile.ExpireDay,
		"Priority":    profile.Priority,
		"Disabled":    profile.Disabled,
		"MaxSize":     profile.MaxSizeMb,
		"Properties":  model.StringInterfaceToJson(profile.Properties),
		"UpdatedAt":   profile.UpdatedAt,
		"UpdatedBy":   profile.UpdatedBy.GetSafeId(),
		"DomainId":    profile.DomainId,
		"Description": profile.Description,
		"Id":          profile.Id,
	})

	if err != nil {
		return nil, model.NewCustomCodeError("store.sql_file_backend_profile.update.app_error", fmt.Sprintf("id=%d, domain=%d, %s", profile.Id, profile.DomainId, err.Error()), extractCodeFromErr(err))
	}

	return profile, nil
}

func (s SqlFileBackendProfileStore) Delete(domainId, id int64) model.AppError {
	if _, err := s.GetMaster().Exec(`delete from storage.file_backend_profiles p where id = :Id and domain_id = :DomainId`,
		map[string]interface{}{"Id": id, "DomainId": domainId}); err != nil {
		return model.NewCustomCodeError("store.sql_file_backend_profile.delete.app_error", fmt.Sprintf("Id=%v, %s", id, err.Error()), extractCodeFromErr(err))
	}
	return nil
}

func (s SqlFileBackendProfileStore) GetById(id int) (*model.FileBackendProfile, model.AppError) {
	var profile *model.FileBackendProfile
	err := s.GetMaster().SelectOne(&profile, `select p.id, call_center.cc_get_lookup(c.id, c.name) as created_by, p.created_at, call_center.cc_get_lookup(u.id, u.name) as updated_by,
       p.updated_at, p.name, p.description, p.expire_day as expire_days, p.priority, p.disabled, p.max_size_mb as max_size, p.properties,
       p.type, coalesce(s.size, 0) data_size, coalesce(s.cnt, 0) data_count, p.domain_id
from storage.file_backend_profiles p
    left join lateral (
        select sum(s.size) size, sum(s.count) as cnt
        from storage.files_statistics s
        where s.profile_id = p.id
    ) s on true
    left join directory.wbt_user c on c.id = p.created_by
    left join directory.wbt_user u on u.id = p.updated_by
    where p.id = :Id `, map[string]interface{}{
		"Id": id,
	})

	if err != nil {
		return nil, model.NewCustomCodeError("store.sql_file_backend_profile.get.app_error", fmt.Sprintf("id=%d, %s", id, err.Error()), extractCodeFromErr(err))
	}

	return profile, nil
}

func (s SqlFileBackendProfileStore) GetSyncTime(domainId int64, id int) (*model.FileBackendProfileSync, model.AppError) {
	var sync *model.FileBackendProfileSync

	err := s.GetReplica().SelectOne(&sync, `select p.updated_at, p.disabled
from storage.file_backend_profiles p
where p.domain_id = :DomainId and p.id = :Id`, map[string]interface{}{
		"DomainId": domainId,
		"Id":       id,
	})

	if err != nil {
		return nil, model.NewCustomCodeError("store.sql_file_backend_profile.sync_tyme.app_error", fmt.Sprintf("id=%d, %s", id, err.Error()), extractCodeFromErr(err))
	}

	return sync, nil
}
