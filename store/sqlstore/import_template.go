package sqlstore

import (
	"fmt"

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

func (s SqlImportTemplateStore) Create(domainId int64, template *model.ImportTemplate) (*model.ImportTemplate, *model.AppError) {
	err := s.GetMaster().SelectOne(&template, `with t as (
    insert into storage.import_template (domain_id, name, description, source_type, source_id, parameters)
    values (:DomainId, :Name, :Description, :SourceType, :SourceId, :Parameters)
    returning *
)
select
    t.id,
    t.name,
    t.description,
    t.source_type,
    t.source_id,
    t.parameters,
    jsonb_build_object('id', s.id, 'name', s.name) as source
from  t
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
	})

	if err != nil {
		return nil, model.NewAppError("SqlImportTemplateStore.Create", "store.sql_import_template.create.app_error", nil,
			fmt.Sprintf("name=%v, %v", template.Name, err.Error()), extractCodeFromErr(err))
	}

	return template, nil
}
func (s SqlImportTemplateStore) GetAllPage(domainId int64, search *model.SearchImportTemplate) ([]*model.ImportTemplate, *model.AppError) {
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
		return nil, model.NewAppError("SqlImportTemplateStore.GetAllPage", "store.sql_import_template.get_all.finding.app_error",
			nil, err.Error(), extractCodeFromErr(err))
	}

	return templates, nil
}
func (s SqlImportTemplateStore) Get(domainId int64, id int32) (*model.ImportTemplate, *model.AppError) {
	var template *model.ImportTemplate
	err := s.GetReplica().SelectOne(&template, `select
    t.id,
    t.name,
    t.description,
    t.source_type,
    t.source_id,
    t.parameters,
    jsonb_build_object('id', s.id, 'name', s.name) as source
from storage.import_template t
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
		return nil, model.NewAppError("SqlImportTemplateStore.Get", "store.sql_import_template.get.app_error",
			nil, err.Error(), extractCodeFromErr(err))
	}

	return template, nil

}
func (s SqlImportTemplateStore) Update(domainId int64, template *model.ImportTemplate) (*model.ImportTemplate, *model.AppError) {
	err := s.GetMaster().SelectOne(&template, `with t as (
    update storage.import_template t
    set name = :Name,
        description = :Description,
        parameters = :Parameters,
        source_type = :SourceType,
        source_id = :SourceId
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
    jsonb_build_object('id', s.id, 'name', s.name) as source
from t
    left join lateral (
        select q.id, q.name
        from call_center.cc_queue q
        where q.id = t.source_id and q.domain_id = t.domain_id
        limit 1
    ) s on true;`, map[string]interface{}{
		"DomainId":    domainId,
		"Id":          template.Id,
		"Name":        template.Name,
		"Description": template.Description,
		"Parameters":  model.StringInterfaceToJson(template.Parameters),
		"SourceType":  template.SourceType,
		"SourceId":    template.SourceId,
	})

	if err != nil {
		return nil, model.NewAppError("SqlImportTemplateStore.Update", "store.sql_import_template.update.app_error",
			nil, err.Error(), extractCodeFromErr(err))
	}

	return template, nil
}
func (s SqlImportTemplateStore) Delete(domainId int64, id int32) *model.AppError {
	if _, err := s.GetMaster().Exec(`delete from storage.import_template t where id = :Id and domain_id = :DomainId`,
		map[string]interface{}{"Id": id, "DomainId": domainId}); err != nil {
		return model.NewAppError("SqlImportTemplateStore.Delete", "store.sql_import_template.delete.app_error", nil,
			fmt.Sprintf("Id=%v, %s", id, err.Error()), extractCodeFromErr(err))
	}
	return nil
}
