package model

import (
	"encoding/json"
	"io"

	engine "github.com/webitel/engine/model"
)

const (
	JOB_TYPE_DATA_RETENTION    = "data_retention"
	JOB_TYPE_DELETE_RECORDINGS = "delete_recordings"

	JOB_TYPE_SYNC_FILES = "sync_files"

	JOB_STATUS_PENDING          = "pending"
	JOB_STATUS_IN_PROGRESS      = "in_progress"
	JOB_STATUS_SUCCESS          = "success"
	JOB_STATUS_ERROR            = "error"
	JOB_STATUS_CANCEL_REQUESTED = "cancel_requested"
	JOB_STATUS_CANCELED         = "canceled"
)

type Job struct {
	Id             string            `db:"id" json:"id"`
	Type           string            `db:"type" json:"type"`
	Priority       int64             `db:"priority" json:"priority"`
	ScheduleId     *int64            `db:"schedule_id" json:"schedule_id"`
	ScheduleTime   int64             `db:"schedule_time" json:"schedule_time"`
	CreateAt       int64             `db:"create_at" json:"create_at"`
	StartAt        int64             `db:"start_at" json:"start_at"`
	LastActivityAt int64             `db:"last_activity_at" json:"last_activity_at"`
	Status         string            `db:"status" json:"status"`
	Progress       int64             `db:"progress" json:"progress"`
	Data           map[string]string `db:"data" json:"data"`
}

func (j *Job) IsValid() engine.AppError {
	if len(j.Id) != 26 {
		return engine.NewBadRequestError("model.job.is_valid.id.app_error", "id="+j.Id)
	}

	if j.CreateAt == 0 {
		return engine.NewBadRequestError("model.job.is_valid.create_at.app_error", "id="+j.Id)
	}

	switch j.Type {
	case JOB_TYPE_SYNC_FILES:
	default:
		return engine.NewBadRequestError("model.job.is_valid.type.app_error", "id="+j.Id)
	}

	switch j.Status {
	case JOB_STATUS_PENDING:
	case JOB_STATUS_IN_PROGRESS:
	case JOB_STATUS_SUCCESS:
	case JOB_STATUS_ERROR:
	case JOB_STATUS_CANCEL_REQUESTED:
	case JOB_STATUS_CANCELED:
	default:
		return engine.NewBadRequestError("model.job.is_valid.status.app_error", "id="+j.Id)
	}

	return nil
}

func (js *Job) ToJson() string {
	b, _ := json.Marshal(js)
	return string(b)
}

func JobFromJson(data io.Reader) *Job {
	var job Job
	if err := json.NewDecoder(data).Decode(&job); err == nil {
		return &job
	} else {
		return nil
	}
}

func JobsToJson(jobs []*Job) string {
	b, _ := json.Marshal(jobs)
	return string(b)
}

func JobsFromJson(data io.Reader) []*Job {
	var jobs []*Job
	if err := json.NewDecoder(data).Decode(&jobs); err == nil {
		return jobs
	} else {
		return nil
	}
}

func (js *Job) DataToJson() string {
	b, _ := json.Marshal(js.Data)
	return string(b)
}

type Worker interface {
	Run()
	Stop()
	JobChannel() chan<- Job
}

type Scheduler interface {
	Name() string
	JobType() string
	Enabled(cfg *Config) bool
	ScheduleJob(cfg *Config, pendingJobs bool, lastSuccessfulJob *Job) (*Job, engine.AppError)
}
