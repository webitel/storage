package sqlstore

import (
	"context"
	"fmt"
	"github.com/lib/pq"
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

func (s *SqlFilePoliciesStore) Create(ctx context.Context, domainId int64, policy *model.FilePolicy) (*model.FilePolicy, engine.AppError) {
	err := s.GetMaster().WithContext(ctx).SelectOne(&policy, `with p as (
    insert into storage.file_policies (domain_id, created_at, created_by, updated_at, updated_by, name, enabled, mime_types,
                                       speed_download, speed_upload, description, channels, retention_days)
    values (:DomainId, :CreatedAt, :CreatedBy, :UpdatedAt, :UpdatedBy, :Name, :Enabled, :MimeTypes,
            :SpeedDownload, :SpeedUpload, :Description, :Channels, :RetentionDays)
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
       p.speed_upload,
       p.retention_days		
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
		"RetentionDays": policy.RetentionDays,
	})

	if err != nil {
		return nil, engine.NewCustomCodeError("store.sql_file_policy.create.app_error", fmt.Sprintf("name=%v, %v", policy.Name, err.Error()), extractCodeFromErr(err))
	}

	return policy, nil
}

func (s *SqlFilePoliciesStore) GetAllPage(ctx context.Context, domainId int64, search *model.SearchFilePolicy) ([]*model.FilePolicy, engine.AppError) {
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

func (s *SqlFilePoliciesStore) Get(ctx context.Context, domainId int64, id int32) (*model.FilePolicy, engine.AppError) {
	var policy *model.FilePolicy
	err := s.GetMaster().WithContext(ctx).SelectOne(&policy, `SELECT p.id,
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
       p.speed_upload,
       p.retention_days
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

func (s *SqlFilePoliciesStore) Update(ctx context.Context, domainId int64, policy *model.FilePolicy) (*model.FilePolicy, engine.AppError) {
	err := s.GetMaster().WithContext(ctx).SelectOne(&policy, `with p as (
    update storage.file_policies
        set updated_at = :UpdatedAt,
            updated_by = :UpdatedBy,
            enabled = :Enabled,
            name = :Name,
            description = :Description,
            speed_upload = :SpeedUpload,
            speed_download = :SpeedDownload,
            mime_types = :MimeTypes,
            channels = :Channels,
			retention_days = :RetentionDays
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
       p.speed_upload,
	   p.retention_days
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
		"RetentionDays": policy.RetentionDays,

		"DomainId": domainId,
		"Id":       policy.Id,
	})

	if err != nil {
		return nil, engine.NewCustomCodeError("store.sql_file_policy.update.app_error", fmt.Sprintf("id=%d, %s", policy.Id, err.Error()), extractCodeFromErr(err))
	}

	return policy, nil
}

func (s *SqlFilePoliciesStore) Delete(ctx context.Context, domainId int64, id int32) engine.AppError {
	if _, err := s.GetMaster().WithContext(ctx).Exec(`delete from storage.file_policies p where id = :Id and domain_id = :DomainId`,
		map[string]interface{}{"Id": id, "DomainId": domainId}); err != nil {
		return engine.NewCustomCodeError("store.sql_file_policy.delete.app_error", fmt.Sprintf("Id=%v, %s", id, err.Error()), extractCodeFromErr(err))
	}
	return nil
}

func (s *SqlFilePoliciesStore) ChangePosition(ctx context.Context, domainId int64, fromId, toId int32) engine.AppError {
	i, err := s.GetMaster().WithContext(ctx).SelectInt(`with t as (
		select p.id,
           case when p.position > lead(p.position) over () then lead(p.position) over () else lag(p.position) over (order by p.position desc) end as new_pos,
           count(*) over () cnt
        from storage.file_policies p
        where p.id in (:FromId, :ToId) and p.domain_id = :DomainId
        order by p.position desc
	),
	u as (
		update storage.file_policies u
		set position = t.new_pos
		from t
		where t.id = u.id and t.cnt = 2 and  :FromId != :ToId
		returning u.id
	)
	select count(*)
	from u`, map[string]interface{}{
		"FromId":   fromId,
		"ToId":     toId,
		"DomainId": domainId,
	})

	if err != nil {
		return engine.NewCustomCodeError("store.sql_file_policy.change_position.app_error", fmt.Sprintf("FromId=%v, ToId=%v %s", fromId, toId, err.Error()), extractCodeFromErr(err))
	}

	if i == 0 {
		return engine.NewNotFoundError("store.sql_file_policy.change_position.not_found", fmt.Sprintf("FromId=%v, ToId=%v", fromId, toId))
	}

	return nil
}

func (s *SqlFilePoliciesStore) AllByDomainId(ctx context.Context, domainId int64) ([]model.FilePolicy, engine.AppError) {
	var list []model.FilePolicy
	_, err := s.GetReplica().WithContext(ctx).Select(&list, `select id, channels, mime_types, p.name, p.speed_download, p.speed_upload, p.retention_days, max(updated_at) over ()
from storage.file_policies p
where p.domain_id = :DomainId
    and p.enabled
order by position desc;`, map[string]interface{}{
		"DomainId": domainId,
	})

	if err != nil {
		return nil, engine.NewCustomCodeError("store.sql_file_policy.all_by_domain.app_error", err.Error(), extractCodeFromErr(err))
	}

	return list, nil
}

func (s *SqlFilePoliciesStore) SetRetentionDay(ctx context.Context, domainId int64, policy *model.FilePolicy) (int64, engine.AppError) {
	m := make([]string, 0, len(policy.MimeTypes))
	for _, v := range policy.MimeTypes {
		m = append(m, policyMaskToLike(v))
	}

	res, err := s.GetMaster().WithContext(ctx).Exec(`update storage.files
set retention_until = uploaded_at + (:RetentionDays || 'days')::interval
where domain_id = :DomainId
    and channel = any(:Channels::varchar[])
    and mime_type ilike any (:Mime::varchar[])`, map[string]any{
		"DomainId":      domainId,
		"Channels":      pq.Array(policy.Channels),
		"Mime":          pq.Array(m),
		"RetentionDays": policy.RetentionDays,
	})

	if err != nil {
		return 0, engine.NewCustomCodeError("store.sql_file_policy.apply.app_error", err.Error(), extractCodeFromErr(err))
	}

	u, err := res.RowsAffected()
	if err != nil {
		return 0, engine.NewCustomCodeError("store.sql_file_policy.apply.app_error", err.Error(), extractCodeFromErr(err))
	}

	return u, nil
}

func policyMaskToLike(s string) string {
	out := []rune(s)

	for k, v := range s {
		switch v {
		case '*':
			out[k] = '%'
		case '?':
			out[k] = '_'
		default:
			out[k] = v

		}
	}

	return string(out)
}
