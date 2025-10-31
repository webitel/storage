package model

import (
	"encoding/json"
	"fmt"
	"time"
)

type SearchFile struct {
	ListRequest
	Ids            []int64
	UploadedAt     *FilterBetween
	UploadedBy     []int64
	ReferenceIds   []string // todo uuid rename
	Channels       []string
	RetentionUntil *FilterBetween
	Removed        *bool
	AgentIds       []int
}

type CustomFileProperties struct {
	StartTime int `json:"start_time,omitempty"`
	EndTime   int `json:"end_time,omitempty"`
	Width     int `json:"width,omitempty"`
	Height    int `json:"height,omitempty"`
}

type BaseFile struct {
	Name             string                `db:"name" json:"name"`
	ViewName         *string               `db:"view_name" json:"view_name,omitempty"`
	Size             int64                 `db:"size" json:"size"`
	MimeType         string                `db:"mime_type" json:"mime_type"`
	Properties       StringInterface       `db:"properties" json:"properties"`
	SHA256Sum        *string               `db:"sha256sum" json:"sha256sum,omitempty"`
	Instance         string                `db:"instance" json:"-"`
	Channel          *string               `db:"channel" json:"channel"`
	RetentionUntil   *time.Time            `db:"retention_until" json:"retention_until"`
	UploadedBy       *Lookup               `db:"uploaded_by" json:"uploaded_by"`
	Malware          *MalwareScan          `db:"malware" json:"malware,omitempty"`
	CustomProperties *CustomFileProperties `db:"custom_properties" json:"custom_properties"`
}

type MalwareScan struct {
	Found      bool       `db:"found" json:"found"`
	Status     string     `db:"status" json:"status"`
	Desc       *string    `db:"description" json:"description,omitempty"`
	ScanDate   *time.Time `db:"scan_date" json:"scan_date,omitempty"`
	Quarantine bool       `db:"-" json:"-"`
}

type File struct {
	BaseFile
	Id          int64      `db:"id" json:"id"`
	DomainId    int64      `db:"domain_id" json:"domain_id"`
	Uuid        string     `db:"uuid" json:"uuid"`
	ProfileId   *int       `db:"profile_id" json:"profile_id"`
	CreatedAt   int64      `db:"created_at" json:"created_at"`
	UploadedAt  *time.Time `db:"uploaded_at" json:"uploaded_at"`
	Removed     *bool      `db:"removed" json:"-"`
	NotExists   *bool      `db:"not_exists" json:"-"`
	Safe        bool       `db:"-" json:"-"`
	Thumbnail   *Thumbnail `db:"thumbnail" json:"thumbnail"`
	ReferenceId *string    `db:"reference_id" json:"reference_id"`
	Profile     *Lookup    `db:"profile" json:"profile"`
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

func (t *MalwareScan) ToJson() *[]byte {
	if t == nil {
		return nil
	}

	d, _ := json.Marshal(t)
	return &d
}

func (t *CustomFileProperties) ToJson() *[]byte {
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
	if f.ViewName != nil && *f.ViewName != "" {
		return *f.ViewName
	}

	return f.Name
}

func (f *BaseFile) IsEncrypted() bool {
	if f.Properties != nil {
		if v, ok := f.Properties["encrypted"]; ok {
			ok, _ = v.(bool)
			return ok
		}
	}

	return false
}

func (f *BaseFile) SetEncrypted(encrypted bool) {
	if f.Properties == nil {
		f.Properties = StringInterface{}
	}
	f.Properties["encrypted"] = encrypted
}

func (f *BaseFile) SetMalwareScan(ms MalwareScan) {
	f.Malware = &ms
}

func (f *BaseFile) StringMalware() string {
	if f.Malware == nil {
		return "false"
	}

	desc := "empty"
	if f.Malware.Desc != nil {
		desc = *f.Malware.Desc
	}

	return fmt.Sprintf("true (%s/%s)", f.Malware.Status, desc)
}

func (f *BaseFile) IsQuarantine() bool {
	return f.Malware != nil && f.Malware.Quarantine
}

func (f *BaseFile) SetPolicyId(id int) {
	if f.Properties == nil {
		f.Properties = StringInterface{}
	}
	f.Properties["policy_id"] = id
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
	if f.Properties == nil {
		f.Properties = StringInterface{}
	}
	f.Properties[name] = value
}

func (f *File) GetStoreName() string {
	return f.Name // need uuid ?
}

func (f File) DefaultOrder() string {
	return "uploaded_at"
}

func (f File) AllowFields() []string {
	return []string{"id", "name", "view_name", "size", "mime_type", "reference_id", "profile", "uploaded_at",
		"sha256sum", "channel", "thumbnail", "retention_until", "domain_id", "uploaded_by",
	}
}

func (f File) DefaultFields() []string {
	return []string{"id", "name", "view_name", "size", "mime_type", "uploaded_at", "channel", "uploaded_by"}
}

func (f File) EntityName() string {
	return "files_list"
}

func (s *SearchFile) IsValid() AppError {
	if s.UploadedAt == nil && len(s.Ids) == 0 && len(s.ReferenceIds) == 0 {
		return NewBadRequestError("model.file.search", "updated_at or ids or reference_id must be set")
	}

	return nil
}
