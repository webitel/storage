package model

var (
	SttCacheSize = 100
	JobCacheSize = 10000
)

type TranscriptOptions struct {
	ProfileId       *int   `json:"profile_id"`
	ProfileSyncTime *int64 `json:"profile_sync_time"`

	Locale  *string  `json:"locale"`
	FileIds []int64  `json:"ids"`
	Uuid    []string `json:"uuid"`
}
