package sqlstore

import (
	"fmt"

	"github.com/webitel/engine/auth_manager"
	engine "github.com/webitel/engine/model"

	"github.com/lib/pq"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/store"
)

type SqlImportTemplateStore struct {
	SqlStore
}

func NewSqlImportTemplateStore(sqlStore SqlStore) store.ImportTemplateStore {
	us := &SqlImportTemplateStore{sqlStore}
	return us
}

func (s SqlImportTemplateStore) CheckAccess(domainId int64, id int32, groups []int, access auth_manager.PermissionAccess) (bool, engine.AppError) {

	res, err := s.GetReplica().SelectNullInt(`select 1
		where exists(
          select 1
          from storage.import_template_acl a
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

func (s SqlImportTemplateStore) Create(domainId int64, template *model.ImportTemplate) (*model.ImportTemplate, engine.AppError) {
	err := s.GetMaster().SelectOne(&template, `with t as (
    insert into storage.import_template (domain_id, name, description, source_type, source_id, parameters, created_at, created_by, updated_at, updated_by)
    values (:DomainId, :Name, :Description, :SourceType, :SourceId, :Parameters, :CreatedAt, :CreatedBy, :UpdatedAt, :UpdatedBy)
    returning *
)
select
    t.id,
    t.name,
    t.description,
    t.source_type,
    t.source_id,
    t.parameters,
    jsonb_build_object('id', s.id, 'name', s.name) as source,
    t.created_at,
    storage.get_lookup(c.id, COALESCE(c.name, c.username::text)::character varying) AS created_by,
    t.updated_at,
    storage.get_lookup(u.id, COALESCE(u.name, u.username::text)::character varying) AS updated_by
from t
    LEFT JOIN directory.wbt_user c ON c.id = t.created_by
    LEFT JOIN directory.wbt_user u ON u.id = t.updated_by
    left join lateral (
        select q.id, q.name
        from call_center.cc_queue q
        where q.id = t.source_id and q.domain_id = t.domain_id
        limit 1
    ) s on true`, map[string]interface{}{
		"DomainId":    domainId,
		"Name":        template.Name,
		"Description": template.Description,
		"SourceType":  template.SourceType,
		"SourceId":    template.SourceId,
		"Parameters":  model.StringInterfaceToJson(template.Parameters),
		"CreatedAt":   template.CreatedAt,
		"CreatedBy":   template.CreatedBy.GetSafeId(),
		"UpdatedAt":   template.UpdatedAt,
		"UpdatedBy":   template.UpdatedBy.GetSafeId(),
	})

	if err != nil {
		return nil, engine.NewCustomCodeError("store.sql_import_template.create.app_error", fmt.Sprintf("name=%v, %v", template.Name, err.Error()), extractCodeFromErr(err))
	}

	return template, nil
}
func (s SqlImportTemplateStore) GetAllPage(domainId int64, search *model.SearchImportTemplate) ([]*model.ImportTemplate, engine.AppError) {
	var templates []*model.ImportTemplate

	f := map[string]interface{}{
		"DomainId": domainId,
		"Ids":      pq.Array(search.Ids),
		"Q":        search.GetQ(),
	}

	err := s.ListQuery(&templates, search.ListRequest,
		`domain_id = :DomainId
				and (:Ids::int[] isnull or id = any(:Ids))
				and (:Q::varchar isnull or (name ilike :Q::varchar ))`,
		model.ImportTemplate{}, f)

	if err != nil {
		return nil, engine.NewCustomCodeError("store.sql_import_template.get_all.finding.app_error", err.Error(), extractCodeFromErr(err))
	}

	return templates, nil
}
func (s SqlImportTemplateStore) GetAllPageByGroups(domainId int64, groups []int, search *model.SearchImportTemplate) ([]*model.ImportTemplate, engine.AppError) {
	var templates []*model.ImportTemplate

	f := map[string]interface{}{
		"DomainId": domainId,
		"Ids":      pq.Array(search.Ids),
		"Q":        search.GetQ(),
		"Groups":   pq.Array(groups),
		"Access":   auth_manager.PERMISSION_ACCESS_READ.Value(),
	}

	err := s.ListQuery(&templates, search.ListRequest,
		`domain_id = :DomainId
				and (:Ids::int[] isnull or id = any(:Ids))
				and (:Q::varchar isnull or (name ilike :Q::varchar ))
				and exists(select 1
				  from storage.import_template_acl a
				  where a.dc = t.domain_id and a.object = t.id and a.subject = any(:Groups::int[]) and a.access&:Access = :Access)
				and (:Ids::int[] isnull or id = any(:Ids))
				and (:Q::varchar isnull or (name ilike :Q::varchar ))`,
		model.ImportTemplate{}, f)

	if err != nil {
		return nil, engine.NewCustomCodeError("store.sql_import_template.get_all.finding.app_error", err.Error(), extractCodeFromErr(err))
	}

	return templates, nil
}
func (s SqlImportTemplateStore) Get(domainId int64, id int32) (*model.ImportTemplate, engine.AppError) {
	var template *model.ImportTemplate
	err := s.GetReplica().SelectOne(&template, `select
    t.id,
    t.name,
    t.description,
    t.source_type,
    t.source_id,
    t.parameters,
    jsonb_build_object('id', s.id, 'name', s.name) as source,
    t.created_at,
    storage.get_lookup(c.id, COALESCE(c.name, c.username::text)::character varying) AS created_by,
    t.updated_at,
    storage.get_lookup(u.id, COALESCE(u.name, u.username::text)::character varying) AS updated_by
from storage.import_template t
    LEFT JOIN directory.wbt_user c ON c.id = t.created_by
    LEFT JOIN directory.wbt_user u ON u.id = t.updated_by
    left join lateral (
        select q.id, q.name
        from call_center.cc_queue q
        where q.id = t.source_id and q.domain_id = t.domain_id
        limit 1
    ) s on true
 where t.domain_id = :DomainId and t.id = :Id`, map[string]interface{}{
		"DomainId": domainId,
		"Id":       id,
	})

	if err != nil {
		return nil, engine.NewCustomCodeError("store.sql_import_template.get.app_error", err.Error(), extractCodeFromErr(err))
	}

	return template, nil

}
func (s SqlImportTemplateStore) Update(domainId int64, template *model.ImportTemplate) (*model.ImportTemplate, engine.AppError) {
	err := s.GetMaster().SelectOne(&template, `with t as (
    update storage.import_template t
    set name = :Name,
        description = :Description,
        parameters = :Parameters,
        source_type = :SourceType,
        source_id = :SourceId,
	    updated_at = :UpdatedAt,
		updated_by = :UpdatedBy
    where t.domain_id = :DomainId and t.id = :Id
    returning *
)
select
    t.id,
    t.name,
    t.description,
    t.source_type,
    t.source_id,
    t.parameters,
    jsonb_build_object('id', s.id, 'name', s.name) as source,
    t.created_at,
    storage.get_lookup(c.id, COALESCE(c.name, c.username::text)::character varying) AS created_by,
    t.updated_at,
    storage.get_lookup(u.id, COALESCE(u.name, u.username::text)::character varying) AS updated_by
from t
    LEFT JOIN directory.wbt_user c ON c.id = t.created_by
    LEFT JOIN directory.wbt_user u ON u.id = t.updated_by
    left join lateral (
        select q.id, q.name
        from call_center.cc_queue q
        where q.id = t.source_id and q.domain_id = t.domain_id
        limit 1
    ) s on true`, map[string]interface{}{
		"DomainId":    domainId,
		"Id":          template.Id,
		"Name":        template.Name,
		"Description": template.Description,
		"Parameters":  model.StringInterfaceToJson(template.Parameters),
		"SourceType":  template.SourceType,
		"SourceId":    template.SourceId,
		"UpdatedBy":   template.UpdatedBy.GetSafeId(),
		"UpdatedAt":   template.UpdatedAt,
	})

	if err != nil {
		return nil, engine.NewCustomCodeError("store.sql_import_template.update.app_error", err.Error(), extractCodeFromErr(err))
	}

	return template, nil
}
func (s SqlImportTemplateStore) Delete(domainId int64, id int32) engine.AppError {
	if _, err := s.GetMaster().Exec(`delete from storage.import_template t where id = :Id and domain_id = :DomainId`,
		map[string]interface{}{"Id": id, "DomainId": domainId}); err != nil {
		return engine.NewCustomCodeError("store.sql_import_template.delete.app_error", fmt.Sprintf("Id=%v, %s", id, err.Error()), extractCodeFromErr(err))
	}
	return nil
}
