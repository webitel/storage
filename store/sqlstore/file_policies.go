package sqlstore

import (
	"fmt"
	"github.com/lib/pq"
	"github.com/webitel/engine/auth_manager"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/store"
)

type SqlFilePoliciesStore struct {
	SqlStore
}

func NewSqlFilePoliciesStore(sqlStore SqlStore) store.FilePoliciesStore {
	us := &SqlFilePoliciesStore{sqlStore}

	return us
}

func (s *SqlFilePoliciesStore) CheckAccess(domainId int64, id int32, groups []int, access auth_manager.PermissionAccess) (bool, engine.AppError) {
	res, err := s.GetReplica().SelectNullInt(`select 1
		where exists(
          select 1
          from storage.file_policies_acl a
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

func (s *SqlFilePoliciesStore) Create(domainId int64, policy *model.FilePolicy) (*model.FilePolicy, engine.AppError) {
	err := s.GetMaster().SelectOne(&policy, `with p as (
    insert into storage.file_policies (domain_id, created_at, created_by, updated_at, updated_by, name, enabled, mime_types,
                                       speed_download, speed_upload, description, channels)
    values (:DomainId, :CreatedAt, :CreatedBy, :UpdatedAt, :UpdatedBy, :Name, :Enabled, :MimeTypes,
            :SpeedDownload, :SpeedUpload, :Description, :Channels)
   returning *
)
SELECT p.id,
       p.created_at,
       storage.get_lookup(c.id, COALESCE(c.name, c.username::text)::character varying) AS created_by,
       p.updated_at,
       storage.get_lookup(u.id, COALESCE(u.name, u.username::text)::character varying) AS updated_by,
       p.enabled,
       p.name,
       p.description,
       p.channels,
       p.mime_types,
       p.speed_download,
       p.speed_upload
FROM p
         LEFT JOIN directory.wbt_user c ON c.id = p.created_by
         LEFT JOIN directory.wbt_user u ON u.id = p.updated_by;`, map[string]interface{}{
		"DomainId":      domainId,
		"CreatedAt":     policy.CreatedAt,
		"UpdatedAt":     policy.UpdatedAt,
		"CreatedBy":     policy.CreatedBy.GetSafeId(),
		"UpdatedBy":     policy.UpdatedBy.GetSafeId(),
		"Name":          policy.Name,
		"Enabled":       policy.Enabled,
		"MimeTypes":     pq.Array(policy.MimeTypes),
		"SpeedDownload": policy.SpeedDownload,
		"SpeedUpload":   policy.SpeedUpload,
		"Description":   policy.Description,
		"Channels":      pq.Array(policy.Channels),
	})

	if err != nil {
		return nil, engine.NewCustomCodeError("store.sql_file_policy.create.app_error", fmt.Sprintf("name=%v, %v", policy.Name, err.Error()), extractCodeFromErr(err))
	}

	return policy, nil
}

func (s *SqlFilePoliciesStore) GetAllPage(domainId int64, search *model.SearchFilePolicy) ([]*model.FilePolicy, engine.AppError) {
	var list []*model.FilePolicy

	f := map[string]interface{}{
		"DomainId": domainId,
		"Ids":      pq.Array(search.Ids),
		"Q":        search.GetQ(),
	}

	err := s.ListQuery(&list, search.ListRequest,
		`domain_id = :DomainId
				and (:Ids::int[] isnull or id = any(:Ids))
				and (:Q::varchar isnull or (name ilike :Q::varchar or description ilike :Q::varchar ))`,
		model.FilePolicy{}, f)

	if err != nil {
		return nil, engine.NewCustomCodeError("store.sql_file_policy.get_all.finding.app_error", err.Error(), extractCodeFromErr(err))
	}

	return list, nil
}

func (s *SqlFilePoliciesStore) GetAllPageByGroups(domainId int64, groups []int, search *model.SearchFilePolicy) ([]*model.FilePolicy, engine.AppError) {
	var list []*model.FilePolicy

	f := map[string]interface{}{
		"DomainId": domainId,
		"Ids":      pq.Array(search.Ids),
		"Q":        search.GetQ(),
		"Groups":   pq.Array(groups),
		"Access":   auth_manager.PERMISSION_ACCESS_READ.Value(),
	}

	err := s.ListQuery(&list, search.ListRequest,
		`domain_id = :DomainId
				and exists(select 1
				  from storage.file_policies_acl a
				  where a.dc = t.domain_id and a.object = t.id and a.subject = any(:Groups::int[]) and a.access&:Access = :Access)
				and (:Ids::int[] isnull or id = any(:Ids))
				and (:Q::varchar isnull or (name ilike :Q::varchar or description ilike :Q::varchar ))`,
		model.FilePolicy{}, f)

	if err != nil {
		return nil, engine.NewCustomCodeError("store.sql_file_policy.get_all.finding.app_error", err.Error(), extractCodeFromErr(err))
	}

	return list, nil
}

func (s *SqlFilePoliciesStore) Get(domainId int64, id int32) (*model.FilePolicy, engine.AppError) {
	var policy *model.FilePolicy
	err := s.GetMaster().SelectOne(&policy, `SELECT p.id,
       p.created_at,
       storage.get_lookup(c.id, COALESCE(c.name, c.username::text)::character varying) AS created_by,
       p.updated_at,
       storage.get_lookup(u.id, COALESCE(u.name, u.username::text)::character varying) AS updated_by,
       p.enabled,
       p.name,
       p.description,
       p.channels,
       p.mime_types,
       p.speed_download,
       p.speed_upload
FROM storage.file_policies p
         LEFT JOIN directory.wbt_user c ON c.id = p.created_by
         LEFT JOIN directory.wbt_user u ON u.id = p.updated_by
where p.domain_id = :DomainId
    and p.id = :Id`, map[string]interface{}{
		"Id":       id,
		"DomainId": domainId,
	})

	if err != nil {
		return nil, engine.NewCustomCodeError("store.sql_file_policy.get.app_error", fmt.Sprintf("id=%d, domain=%d, %s", id, domainId, err.Error()), extractCodeFromErr(err))
	}

	return policy, nil
}

func (s *SqlFilePoliciesStore) Update(domainId int64, policy *model.FilePolicy) (*model.FilePolicy, engine.AppError) {
	err := s.GetMaster().SelectOne(&policy, `with p as (
    update storage.file_policies
        set updated_at = :UpdatedAt,
            updated_by = :UpdatedBy,
            enabled = :Enabled,
            name = :Name,
            description = :Description,
            speed_upload = :SpeedUpload,
            speed_download = :SpeedDownload,
            mime_types = :MimeTypes,
            channels= :Channels
        where domain_id = :DomainId and id = :Id
		returning *
)
SELECT p.id,
       p.created_at,
       storage.get_lookup(c.id, COALESCE(c.name, c.username::text)::character varying) AS created_by,
       p.updated_at,
       storage.get_lookup(u.id, COALESCE(u.name, u.username::text)::character varying) AS updated_by,
       p.enabled,
       p.name,
       p.description,
       p.channels,
       p.mime_types,
       p.speed_download,
       p.speed_upload
FROM p
         LEFT JOIN directory.wbt_user c ON c.id = p.created_by
         LEFT JOIN directory.wbt_user u ON u.id = p.updated_by`, map[string]interface{}{
		"UpdatedAt": policy.UpdatedAt,
		"UpdatedBy": policy.UpdatedBy.GetSafeId(),

		"Enabled":       policy.Enabled,
		"Name":          policy.Name,
		"Description":   policy.Description,
		"SpeedUpload":   policy.SpeedUpload,
		"SpeedDownload": policy.SpeedDownload,
		"MimeTypes":     pq.Array(policy.MimeTypes),
		"Channels":      pq.Array(policy.Channels),

		"DomainId": domainId,
		"Id":       policy.Id,
	})

	if err != nil {
		return nil, engine.NewCustomCodeError("store.sql_file_policy.update.app_error", fmt.Sprintf("id=%d, %s", policy.Id, err.Error()), extractCodeFromErr(err))
	}

	return policy, nil
}

func (s *SqlFilePoliciesStore) Delete(domainId int64, id int32) engine.AppError {
	if _, err := s.GetMaster().Exec(`delete from storage.file_policies p where id = :Id and domain_id = :DomainId`,
		map[string]interface{}{"Id": id, "DomainId": domainId}); err != nil {
		return engine.NewCustomCodeError("store.sql_file_policy.delete.app_error", fmt.Sprintf("Id=%v, %s", id, err.Error()), extractCodeFromErr(err))
	}
	return nil
}
