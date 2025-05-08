package sqlstore

import (
	"context"
	"github.com/lib/pq"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/store"
)

type SqlTranscriptFileStore struct {
	SqlStore
}

func NewSqlTranscriptFileStore(sqlStore SqlStore) store.TranscriptFileStore {
	us := &SqlTranscriptFileStore{sqlStore}
	return us
}

func (s *SqlTranscriptFileStore) Store(t *model.FileTranscript) (*model.FileTranscript, model.AppError) {
	err := s.GetMaster().SelectOne(&t, `with t as (
    insert into storage.file_transcript (file_id, transcript, log, profile_id, locale, phrases, channels, uuid, domain_id)
    select :FileId, :Transcript, :Log, :ProfileId, :Locale, :Phrases, :Channels, f.uuid, f.domain_id
	from storage.files f
	where f.id = :FileId::int8
    returning storage.file_transcript.*
)
select t.id,
       storage.get_lookup(f.id, f.name) as file,
       storage.get_lookup(p.id, p.name) as profile,
       t.transcript,
       t.log,
       t.created_at,
	   t.phrases,
	   t.channels	
from t
    left join storage.files f on f.id = t.file_id
    left join storage.cognitive_profile_services p on p.id = t.profile_id`, map[string]interface{}{
		"FileId":     t.File.Id,
		"Transcript": t.TidyTranscript(),
		"Log":        t.Log,
		"Locale":     t.Locale,
		"ProfileId":  t.Profile.Id,
		"Phrases":    t.JsonPhrases(),
		"Channels":   t.JsonChannels(),
	})

	if err != nil {
		return nil, model.NewCustomCodeError("store.sql_stt_file.store.app_error", err.Error(), extractCodeFromErr(err))
	}

	return t, nil
}

func (s *SqlTranscriptFileStore) CreateJobs(domainId int64, params model.TranscriptOptions) ([]*model.FileTranscriptJob, model.AppError) {
	var jobs []*model.FileTranscriptJob
	_, err := s.GetMaster().Select(&jobs, `with trfiles as (
		select 0 as state,
			   fid.id,
			   p.service,
			   json_build_object('locale', coalesce(:Locale::varchar, (p.properties->'default_locale')::varchar),
				   'profile_id', p.id,
				   'profile_sync_time', (extract(epoch from p.updated_at) * 1000 )::int8) as config
		from (select *
		from storage.cognitive_profile_services p
		where p.domain_id = :DomainId::int8
			and p.enabled
			and p.service = :Service::varchar
			and case when :Id::int4 notnull  then p.id = :Id::int4 else p."default" is true end
		limit 1) p,
			 (select f.id
		from storage.files f
		where f.domain_id = :DomainId::int8
			and (f.id = any((:FileIds)::int8[]) )
			union distinct
			select f.id
			from storage.files f
			where f.domain_id = :DomainId::int8
				and f.uuid = any((:Uuid)::varchar[])
				and not exists(select 1 from storage.file_transcript ft where ft.uuid = f.uuid and ft.file_id = f.id)
		) fid
	),
	delerr as (
		delete
		from storage.file_jobs fj
        USING  trfiles i
		where fj.state = 3 and fj.file_id = i.id
		returning fj.*
	)
	insert into storage.file_jobs (state, file_id, action, config)
	select t.state, t.id, t.service, t.config
	from trfiles t
	    left join  delerr d on d.file_id = t.id
	for update
	returning storage.file_jobs.id,
		storage.file_jobs.file_id,
		(extract(epoch from storage.file_jobs.created_at) * 1000)::int8 as created_at;`, map[string]interface{}{
		"DomainId": domainId,
		"FileIds":  pq.Array(params.FileIds),
		"Uuid":     pq.Array(params.Uuid),
		"Id":       params.ProfileId,
		"Locale":   params.Locale,
		"Service":  model.SyncJobSTT,
	})

	if err != nil {
		return nil, model.NewCustomCodeError("store.sql_stt_file.create.jobs.app_error", err.Error(), extractCodeFromErr(err))
	}

	if len(jobs) == 0 {
		return nil, model.NewNotFoundError("store.sql_stt_file.create.jobs.not_found", "Not found profile")
	}

	return jobs, nil
}

func (s *SqlTranscriptFileStore) GetPhrases(domainId, id int64, search *model.ListRequest) ([]*model.TranscriptPhrase, model.AppError) {
	var phrases []*model.TranscriptPhrase
	_, err := s.GetReplica().Select(&phrases, `select p->'start_sec' as start_sec,
       p->'end_sec' as end_sec,
       p->>'channel' as channel,
       p->>'display' as phrase
from storage.file_transcript t
    left join lateral (
        select p
        from jsonb_array_elements(t.phrases) p
        order by p->'start_sec'
        limit :Limit
        offset :Offset
    ) p on true
where id = :Id
	and domain_id = :DomainId
    and p notnull`, map[string]interface{}{
		"Limit":    search.GetLimit(),
		"Offset":   search.GetOffset(),
		"Id":       id,
		"DomainId": domainId,
	})

	if err != nil {
		return nil, model.NewCustomCodeError("store.sql_stt_file.transcript.phrases.error", err.Error(), extractCodeFromErr(err))
	}

	return phrases, nil
}

func (s *SqlTranscriptFileStore) Delete(domainId int64, ids []int64, uuid []string) ([]int64, model.AppError) {
	var res []int64
	_, err := s.GetMaster().Select(&res, `delete
from storage.file_transcript t
where (id = any(:Ids::int8[]) or uuid = any(:Uuid::varchar[]))
    and t.domain_id = :DomainId::int8
returning t.id`, map[string]interface{}{
		"DomainId": domainId,
		"Ids":      pq.Array(ids),
		"Uuid":     pq.Array(uuid),
	})

	if err != nil {
		return nil, model.NewCustomCodeError("store.sql_stt_file.transcript.delete.error", err.Error(), extractCodeFromErr(err))
	}

	return res, nil
}

func (s *SqlTranscriptFileStore) Put(ctx context.Context, domainId int64, uuid string, tr model.FileTranscript) (int64, model.AppError) {
	res, err := s.GetMaster().WithContext(ctx).SelectInt(`insert into storage.file_transcript(file_id, transcript, created_at, locale, phrases, uuid, domain_id)
values (:FileId, :Transcript, :CreatedAt, :Locale, :Phrases, :Uuid, :DomainId)
on conflict (file_id, coalesce(profile_id, 0), locale)
DO UPDATE SET
        transcript = EXCLUDED.transcript,
        phrases = EXCLUDED.phrases,
        uuid = EXCLUDED.uuid
returning id`, map[string]interface{}{
		"FileId":     tr.File.Id,
		"Transcript": tr.Transcript,
		"CreatedAt":  tr.CreatedAt,
		"Locale":     tr.Locale,
		"Phrases":    tr.JsonPhrases(),
		"Uuid":       uuid,
		"DomainId":   domainId,
	})

	if err != nil {
		return 0, model.NewCustomCodeError("store.sql_stt_file.put.app_error", err.Error(), extractCodeFromErr(err))
	}

	return res, nil
}
