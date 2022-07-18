package sqlstore

import (
	"net/http"

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

func (s SqlTranscriptFileStore) GetByFileId(fileId int64, profileId int64) (*model.FileTranscript, *model.AppError) {
	var t model.FileTranscript
	err := s.GetReplica().SelectOne(&t, `select t.id,
       storage.get_lookup(f.id, f.name) as file,
       storage.get_lookup(p.id, p.name) as profile,
       t.transcript,
       t.log,
       t.locale,
       t.created_at
from storage.file_transcript t
    left join storage.files f on f.id = t.file_id
    left join storage.cognitive_profile_services p on p.id = t.profile_id
where t.id = :Id and t.profile_id = :ProfileId`, map[string]interface{}{
		"Id":        fileId,
		"ProfileId": profileId,
	})

	if err != nil {
		return nil, model.NewAppError("SqlTranscriptFileStore.GetByFileId", "store.sql_stt_file.get.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return &t, nil
}

func (s SqlTranscriptFileStore) Store(t *model.FileTranscript) (*model.FileTranscript, *model.AppError) {
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
		return nil, model.NewAppError("SqlTranscriptFileStore.Store", "store.sql_stt_file.store.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return t, nil
}

func (s SqlTranscriptFileStore) CreateJobs(domainId int64, params model.TranscriptOptions) ([]*model.FileTranscriptJob, *model.AppError) {
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
				and not exists(select 1 from storage.file_transcript ft where ft.uuid = f.uuid)
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
		return nil, model.NewAppError("SqlTranscriptFileStore.CreateJobs", "store.sql_stt_file.create.jobs.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	if len(jobs) == 0 {
		return nil, model.NewAppError("SqlTranscriptFileStore.CreateJobs", "store.sql_stt_file.create.jobs.not_found", nil, "Not found profile", http.StatusNotFound)
	}

	return jobs, nil
}

func (s SqlTranscriptFileStore) GetPhrases(domainId, id int64, search *model.ListRequest) ([]*model.TranscriptPhrase, *model.AppError) {
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
		return nil, model.NewAppError("SqlTranscriptFileStore.GetPhrases", "store.sql_stt_file.transcript.phrases.error", nil, err.Error(), extractCodeFromErr(err))
	}

	return phrases, nil
}

func (s SqlTranscriptFileStore) Delete(domainId int64, ids []int64, uuid []string) ([]int64, *model.AppError) {
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
		return nil, model.NewAppError("SqlTranscriptFileStore.Delete", "store.sql_stt_file.transcript.delete.error", nil, err.Error(), extractCodeFromErr(err))
	}

	return res, nil
}
