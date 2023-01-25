package sqlstore

import (
	"fmt"

	"github.com/lib/pq"
	"github.com/webitel/engine/auth_manager"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/store"
)

type SqlCognitiveProfileStore struct {
	SqlStore
}

func NewSqlCognitiveProfileStore(sqlStore SqlStore) store.CognitiveProfileStore {
	us := &SqlCognitiveProfileStore{sqlStore}
	return us
}

func (s SqlCognitiveProfileStore) CheckAccess(domainId, id int64, groups []int, access auth_manager.PermissionAccess) (bool, *model.AppError) {

	res, err := s.GetReplica().SelectNullInt(`select 1
		where exists(
          select 1
          from storage.cognitive_profile_services_acl a
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

func (s SqlCognitiveProfileStore) Create(profile *model.CognitiveProfile) (*model.CognitiveProfile, *model.AppError) {
	err := s.GetMaster().SelectOne(&profile, `with p as (
    insert into storage.cognitive_profile_services (domain_id, provider, properties, created_at, updated_at, created_by,
                                                    updated_by, enabled, name, description, service, "default")
        values (:DomainId, :Provider, :Properties, :CreatedAt, :UpdatedAt, :CreatedBy, :UpdatedBy,
                :Enabled, :Name, :Description, :Service, :Default)
        returning *
)
SELECT p.id,
       p.domain_id,
       p.provider,
       p.properties,
       p.created_at,
       storage.get_lookup(c.id, COALESCE(c.name, c.username::text)::character varying) AS created_by,
       p.updated_at,
       storage.get_lookup(u.id, COALESCE(u.name, u.username::text)::character varying) AS updated_by,
       p.enabled,
       p.name,
       p.description,
       p.service,
       p."default"
FROM p
         LEFT JOIN directory.wbt_user c ON c.id = p.created_by
         LEFT JOIN directory.wbt_user u ON u.id = p.updated_by`, map[string]interface{}{
		"DomainId":   profile.DomainId,
		"Provider":   profile.Provider,
		"Properties": model.StringInterfaceToJson(profile.Properties),
		"CreatedAt":  profile.CreatedAt,
		"UpdatedAt":  profile.UpdatedAt,
		"CreatedBy":  profile.CreatedBy.GetSafeId(),
		"UpdatedBy":  profile.UpdatedBy.GetSafeId(),

		"Enabled":     profile.Enabled,
		"Name":        profile.Name,
		"Description": profile.Description,
		"Service":     profile.Service,
		"Default":     profile.Default,
	})

	if err != nil {
		return nil, model.NewAppError("SqlCognitiveProfileStore.Create", "store.sql_cognitive_profile_store.create.app_error", nil,
			fmt.Sprintf("name=%v, %v", profile.Name, err.Error()), extractCodeFromErr(err))
	}

	return profile, nil
}

func (s SqlCognitiveProfileStore) GetAllPage(domainId int64, search *model.SearchCognitiveProfile) ([]*model.CognitiveProfile, *model.AppError) {
	var profiles []*model.CognitiveProfile

	f := map[string]interface{}{
		"DomainId": domainId,
		"Ids":      pq.Array(search.Ids),
		"Service":  pq.Array(search.Service),
		"Q":        search.GetQ(),
		"Enabled":  search.Enabled,
	}

	err := s.ListQuery(&profiles, search.ListRequest,
		`domain_id = :DomainId
				and (:Ids::int[] isnull or id = any(:Ids))
			    and (:Service::varchar[] isnull or service = any(:Service))
				and (not :Enabled::bool or enabled)
				and (:Q::varchar isnull or (name ilike :Q::varchar ))`,
		model.CognitiveProfile{}, f)

	if err != nil {
		return nil, model.NewAppError("SqlCognitiveProfileStore.GetAllPage", "store.sql_cognitive_profile_store.get_all.finding.app_error",
			nil, err.Error(), extractCodeFromErr(err))
	}

	return profiles, nil
}

func (s SqlCognitiveProfileStore) GetAllPageByGroups(domainId int64, groups []int, search *model.SearchCognitiveProfile) ([]*model.CognitiveProfile, *model.AppError) {
	var profiles []*model.CognitiveProfile

	f := map[string]interface{}{
		"DomainId": domainId,
		"Ids":      pq.Array(search.Ids),
		"Service":  pq.Array(search.Service),
		"Q":        search.GetQ(),
		"Groups":   pq.Array(groups),
		"Access":   auth_manager.PERMISSION_ACCESS_READ.Value(),
		"Enabled":  search.Enabled,
	}

	err := s.ListQuery(&profiles, search.ListRequest,
		`domain_id = :DomainId
				and exists(select 1
				  from storage.cognitive_profile_services_acl a
				  where a.dc = t.domain_id and a.object = t.id and a.subject = any(:Groups::int[]) and a.access&:Access = :Access)
				and (not :Enabled::bool or enabled)
				and (:Ids::int[] isnull or id = any(:Ids))
				and (:Service::varchar[] isnull or service = any(:Service))
				and (:Q::varchar isnull or (name ilike :Q::varchar ))`,
		model.CognitiveProfile{}, f)

	if err != nil {
		return nil, model.NewAppError("SqlCognitiveProfileStore.GetAllPageByGroups", "store.sql_cognitive_profile_store.get_all.finding.app_error",
			nil, err.Error(), extractCodeFromErr(err))
	}

	return profiles, nil
}

func (s SqlCognitiveProfileStore) Get(id, domainId int64) (*model.CognitiveProfile, *model.AppError) {
	var profile *model.CognitiveProfile
	err := s.GetMaster().SelectOne(&profile, `SELECT p.id,
       p.domain_id,
       p.provider,
       p.properties,
       p.created_at,
       storage.get_lookup(c.id, COALESCE(c.name, c.username::text)::character varying) AS created_by,
       p.updated_at,
       storage.get_lookup(u.id, COALESCE(u.name, u.username::text)::character varying) AS updated_by,
       p.enabled,
       p.name,
       p.description,
       p.service,
       p."default"
FROM storage.cognitive_profile_services p
         LEFT JOIN directory.wbt_user c ON c.id = p.created_by
         LEFT JOIN directory.wbt_user u ON u.id = p.updated_by
where p.id = :Id and p.domain_id = :DomainId`, map[string]interface{}{
		"Id":       id,
		"DomainId": domainId,
	})

	if err != nil {
		return nil, model.NewAppError("SqlCognitiveProfileStore.Get", "store.sql_cognitive_profile_store.get.app_error", nil,
			fmt.Sprintf("id=%d, domain=%d, %s", id, domainId, err.Error()), extractCodeFromErr(err))
	}

	return profile, nil
}

func (s SqlCognitiveProfileStore) Update(profile *model.CognitiveProfile) (*model.CognitiveProfile, *model.AppError) {
	err := s.GetMaster().SelectOne(&profile, `with p as (
    update storage.cognitive_profile_services
        set provider = :Provider,
            properties = :Properties,
            updated_at = :UpdatedAt,
            updated_by = :UpdatedBy,
            enabled = :Enabled,
            name = :Name,
            description = :Description,
            service = :Service,
            "default" = :Default
        where id = :Id and domain_id = :DomainId
        returning *
)
SELECT p.id,
       p.domain_id,
       p.provider,
       p.properties,
       p.created_at,
       storage.get_lookup(c.id, COALESCE(c.name, c.username::text)::character varying) AS created_by,
       p.updated_at,
       storage.get_lookup(u.id, COALESCE(u.name, u.username::text)::character varying) AS updated_by,
       p.enabled,
       p.name,
       p.description,
       p.service,
       p."default"
FROM p
         LEFT JOIN directory.wbt_user c ON c.id = p.created_by
         LEFT JOIN directory.wbt_user u ON u.id = p.updated_by`, map[string]interface{}{
		"Provider":   profile.Provider,
		"Properties": model.StringInterfaceToJson(profile.Properties),
		"UpdatedAt":  profile.UpdatedAt,
		"UpdatedBy":  profile.UpdatedBy.GetSafeId(),

		"Enabled":     profile.Enabled,
		"Name":        profile.Name,
		"Description": profile.Description,
		"Service":     profile.Service,
		"Default":     profile.Default,

		"DomainId": profile.DomainId,
		"Id":       profile.Id,
	})

	if err != nil {
		return nil, model.NewAppError("SqlCognitiveProfileStore.Update", "store.sql_cognitive_profile_store.update.app_error", nil,
			fmt.Sprintf("id=%d, domain=%d, %s", profile.Id, profile.DomainId, err.Error()), extractCodeFromErr(err))
	}

	return profile, nil
}

func (s SqlCognitiveProfileStore) Delete(domainId, id int64) *model.AppError {
	if _, err := s.GetMaster().Exec(`delete from storage.cognitive_profile_services p where id = :Id and domain_id = :DomainId`,
		map[string]interface{}{"Id": id, "DomainId": domainId}); err != nil {
		return model.NewAppError("SqlCognitiveProfileStore.Delete", "store.sql_cognitive_profile_store.delete.app_error", nil,
			fmt.Sprintf("Id=%v, %s", id, err.Error()), extractCodeFromErr(err))
	}
	return nil
}

func (s SqlCognitiveProfileStore) GetById(id int64) (*model.CognitiveProfile, *model.AppError) {
	var profile *model.CognitiveProfile
	err := s.GetMaster().SelectOne(&profile, `SELECT p.id,
       p.domain_id,
       p.provider,
       p.properties,
       p.created_at,
       storage.get_lookup(c.id, COALESCE(c.name, c.username::text)::character varying) AS created_by,
       p.updated_at,
       storage.get_lookup(u.id, COALESCE(u.name, u.username::text)::character varying) AS updated_by,
       p.enabled,
       p.name,
       p.description,
       p.service,
       p."default"
FROM storage.cognitive_profile_services p
         LEFT JOIN directory.wbt_user c ON c.id = p.created_by
         LEFT JOIN directory.wbt_user u ON u.id = p.updated_by
where p.id = :Id`, map[string]interface{}{
		"Id": id,
	})

	if err != nil {
		return nil, model.NewAppError("SqlCognitiveProfileStore.GetById", "store.sql_cognitive_profile_store.get_by_id.app_error", nil,
			fmt.Sprintf("id=%d, %s", id, err.Error()), extractCodeFromErr(err))
	}

	return profile, nil
}

func (s SqlCognitiveProfileStore) SearchTtsProfile(domainId int64, profileId int) (*model.TtsProfile, *model.AppError) {
	var profile *model.TtsProfile
	err := s.GetMaster().SelectOne(&profile, `select p.enabled, p.provider, p.properties
from storage.cognitive_profile_services p
where p.domain_id = :DomainId::int8
    and case when :Id::int = 0 then p."default" else p.id = :Id::int end
    and p.service = 'TTS'`, map[string]interface{}{
		"DomainId": domainId,
		"Id":       profileId,
	})

	if err != nil {
		return nil, model.NewAppError("SqlCognitiveProfileStore.SearchTtsProfile", "store.sql_cognitive_profile_store.get_tts.app_error", nil,
			fmt.Sprintf("profileId=%d, %s", profileId, err.Error()), extractCodeFromErr(err))
	}

	return profile, nil
}
