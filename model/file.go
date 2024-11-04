package model

import (
	"encoding/json"
	"time"
)

type BaseFile struct {
	Name           string          `db:"name" json:"name"`
	ViewName       *string         `db:"view_name" json:"view_name,omitempty"`
	Size           int64           `db:"size" json:"size"`
	MimeType       string          `db:"mime_type" json:"mime_type"`
	Properties     StringInterface `db:"properties" json:"properties"`
	SHA256Sum      *string         `db:"sha256sum" json:"sha256sum,omitempty"`
	Instance       string          `db:"instance" json:"-"`
	Channel        *string         `db:"channel" json:"channel"`
	RetentionUntil *time.Time      `db:"retention_until" json:"retention_until"`
}

type File struct {
	BaseFile
	Id        int64      `db:"id" json:"id"`
	DomainId  int64      `db:"domain_id" json:"domain_id"`
	Uuid      string     `db:"uuid" json:"uuid"`
	ProfileId *int       `db:"profile_id" json:"profile_id"`
	CreatedAt int64      `db:"created_at" json:"created_at"`
	Removed   *bool      `db:"removed" json:"-"`
	NotExists *bool      `db:"not_exists" json:"-"`
	Safe      bool       `db:"-" json:"-"`
	Thumbnail *Thumbnail `db:"thumbnail" json:"thumbnail"`
}

type Thumbnail struct {
	BaseFile
	Scale string `json:"scale"`
}

func (t *Thumbnail) ToJson() *[]byte {
	if t == nil {
		return nil
	}

	d, _ := json.Marshal(t)
	return &d
}

func (f *File) Domain() int64 {
	return f.DomainId
}

func (f *BaseFile) GetSize() int64 {
	return f.Size
}

func (f *BaseFile) GetMimeType() string {
	return f.MimeType
}

func (f *BaseFile) GetViewName() string {
	if f.ViewName != nil {
		return *f.ViewName
	}

	return f.Name
}

func (f *BaseFile) GetChannel() *string {

	return f.Channel
}

type RemoveFile struct {
	Id        int    `db:"id"`
	FileId    int64  `db:"file_id"`
	CreatedAt int64  `db:"created_at"`
	CreatedBy string `db:"created_by"`
}

func (self *RemoveFile) PreSave() {
	self.CreatedAt = GetMillis()
}

type RemoveFileJob struct {
	File
	RemoveFile
}

type FileWithProfile struct {
	File
	ProfileUpdatedAt *int64 `db:"profile_updated_at"`
}

func (f *File) ToJson() string {
	b, _ := json.Marshal(f)
	return string(b)
}

func FileListToJson(list []*File) string {
	b, _ := json.Marshal(list)
	return string(b)
}

func (self BaseFile) GetPropertyString(name string) string {
	return self.Properties.GetString(name)
}

func (self BaseFile) SetPropertyString(name, value string) {
	self.Properties[name] = value
}

func (self File) GetStoreName() string {
	return self.Name // need uuid ?
}
