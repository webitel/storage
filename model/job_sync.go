package model

const (
	SyncJobRemove = "remove"
	SyncJobSTT    = "STT"
	Transcoding   = "transcoding"
	Restore       = "restore"
)

type SyncJob struct {
	BaseFile
	Id               int64  `json:"id" db:"id"`
	FileId           int64  `json:"file_id" db:"file_id"`
	DomainId         int64  `json:"domain_id" db:"domain_id"`
	ProfileId        *int   `json:"profile_id" db:"profile_id"`
	ProfileUpdatedAt *int64 `json:"profile_updated_at" db:"profile_updated_at"`
	Action           string `json:"action" db:"action"`
	Log              []byte `json:"log" db:"log"`
	Config           []byte `json:"config" db:"config"`
}
