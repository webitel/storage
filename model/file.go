package model

import (
	"encoding/json"
	engine "github.com/webitel/engine/model"
	"time"
)

type SearchFile struct {
	ListRequest
	Ids            []int64
	UploadedAt     *FilterBetween
	UploadedBy     []int64
	ReferenceIds   []string
	Channels       []string
	RetentionUntil *FilterBetween
}

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
	UploadedBy     *Lookup         `db:"uploaded_by" json:"uploaded_by"`
}

type File struct {
	BaseFile
	Id         int64      `db:"id" json:"id"`
	DomainId   int64      `db:"domain_id" json:"domain_id"`
	Uuid       string     `db:"uuid" json:"uuid"`
	ProfileId  *int       `db:"profile_id" json:"profile_id"`
	CreatedAt  int64      `db:"created_at" json:"created_at"`
	UploadedAt *time.Time `db:"uploaded_at" json:"uploaded_at"`
	Removed    *bool      `db:"removed" json:"-"`
	NotExists  *bool      `db:"not_exists" json:"-"`
	Safe       bool       `db:"-" json:"-"`
	Thumbnail  *Thumbnail `db:"thumbnail" json:"thumbnail"`
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

func (f *RemoveFile) PreSave() {
	f.CreatedAt = GetMillis()
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

func (f *BaseFile) GetPropertyString(name string) string {
	return f.Properties.GetString(name)
}

func (f *BaseFile) SetPropertyString(name, value string) {
	f.Properties[name] = value
}

func (f *File) GetStoreName() string {
	return f.Name // need uuid ?
}

func (f File) DefaultOrder() string {
	return "uploaded_by"
}

func (f File) AllowFields() []string {
	return []string{"id", "name", "view_name", "size", "mime_type", "reference_id", "profile", "uploaded_at", "updated_by",
		"sha256sum", "channel", "thumbnail", "retention_until",
	}
}

func (f File) DefaultFields() []string {
	return []string{"id", "name", "view_name", "size", "mime_type", "uploaded_at", "channel"}
}

func (f File) EntityName() string {
	return "files_list"
}

func (s *SearchFile) IsValid() engine.AppError {
	if s.UploadedAt == nil && len(s.Ids) == 0 && len(s.ReferenceIds) == 0 {
		return engine.NewBadRequestError("model.file.search", "updated_at or ids or reference_id must be set")
	}

	return nil
}
