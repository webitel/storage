package model

import (
	"sync"
)

const (
	UploadFileChannelUnknown       = "unknown"
	UploadFileChannelCall          = "call"
	UploadFileChannelChat          = "chat"
	UploadFileChannelMail          = "mail"
	UploadFileChannelMedia         = "media"
	UploadFileChannelLog           = "log"
	UploadFileChannelKnowledgebase = "knowledgebase"
	UploadFileChannelCase          = "case"
	//UploadFileChannelScreenshot    = "screenshot"
	UploadFileChannelScreenRecording = "screenrecording"
)

type JobUploadFile struct {
	BaseFile

	Id        int64      `db:"id"`
	State     int        `db:"state"`
	Uuid      string     `db:"uuid"`
	DomainId  int64      `db:"domain_id"`
	EmailMsg  string     `db:"email_msg"`
	EmailSub  string     `db:"email_sub"`
	CreatedAt int64      `db:"created_at"`
	UpdatedAt int64      `db:"updated_at"`
	Attempts  int        `db:"attempts,default:0" json:"attempts"`
	Thumbnail *Thumbnail `db:"thumbnail" json:"thumbnail"`

	GenerateThumbnail bool `db:"-"`

	mu sync.RWMutex `db:"-" json:"-"`
}

type JobUploadFileWithProfile struct {
	JobUploadFile
	ProfileId        *int   `json:"profile_id" db:"profile_id"`
	ProfileUpdatedAt *int64 `json:"profile_updated_at" db:"profile_updated_at"`
}

func (f *JobUploadFile) PreSave() {
	if f.CreatedAt == 0 {
		f.CreatedAt = GetMillis()
	}
	f.UpdatedAt = GetMillis()
}

func (f *JobUploadFile) GetSize() int64 {
	return f.Size
}

func (f *JobUploadFile) GetMimeType() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.MimeType
}

func (f *JobUploadFile) SetMimeType(mimeType string) {
	f.mu.Lock()
	f.MimeType = mimeType
	f.mu.Unlock()
}

func (f *JobUploadFile) GetViewName() string {
	if f.ViewName != nil {
		return *f.ViewName
	}

	return f.Name
}

func (f *JobUploadFile) GetChannel() *string {
	return f.Channel
}

// TODO
func (f *JobUploadFile) GetPropertyString(name string) string {
	return ""
}
func (f *JobUploadFile) SetPropertyString(name, value string) {
	f.BaseFile.SetPropertyString(name, value)
}
func (f *JobUploadFile) Domain() int64 {
	return f.DomainId
}
