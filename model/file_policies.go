package model

import (
	engine "github.com/webitel/engine/model"
	"time"
)

var (
	PolicyErrorMaxLimit      = engine.NewForbiddenError("policy.file.allow", "max size")
	PolicyErrorExtUnknown    = engine.NewForbiddenError("policy.file.allow", "extension of file is unknown")
	PolicyErrorExtSuspicious = engine.NewForbiddenError("policy.file.allow", "actual file extension doesn't match declared Content-Type")
	PolicyErrorExtNotAllowed = engine.NewForbiddenError("policy.file.allow", "file extension is not allowed")
)

type FilePolicy struct {
	Id        int32      `json:"id" db:"id"`
	CreatedAt *time.Time `json:"created_at" db:"created_at"`
	CreatedBy *Lookup    `json:"created_by" db:"created_by"`
	UpdatedAt *time.Time `json:"updated_at" db:"updated_at"`
	UpdatedBy *Lookup    `json:"updated_by" db:"updated_by"`

	Name          string      `json:"name" db:"name"`
	Description   string      `json:"description" db:"description"`
	Enabled       bool        `json:"enabled" db:"enabled"`
	MimeTypes     StringArray `json:"mime_types" db:"mime_types"`
	Channels      StringArray `json:"channels" db:"channels"`
	SpeedDownload int64       `json:"speed_download" db:"speed_download"`
	SpeedUpload   int64       `json:"speed_upload" db:"speed_upload"`
	MaxUploadSize int64       `json:"max_upload_size" db:"max_upload_size"`
	RetentionDays int32       `json:"retention_days" db:"retention_days"`
	Position      int32       `json:"position" db:"position"`
	Max           *time.Time  `json:"max" db:"max"`
}

type FilePolicyPath struct {
	UpdatedBy Lookup
	UpdatedAt time.Time

	Name          *string     `json:"name" db:"name"`
	Description   *string     `json:"description" db:"description"`
	Enabled       *bool       `json:"enabled" db:"enabled"`
	MimeTypes     StringArray `json:"mime_types" db:"mime_types"`
	Channels      StringArray `json:"channels" db:"channels"`
	SpeedDownload *int64      `json:"speed_download" db:"speed_download"`
	SpeedUpload   *int64      `json:"speed_upload" db:"speed_upload"`
	RetentionDays *int32      `json:"retention_days" db:"retention_days"`
	MaxUploadSize *int64      `json:"max_upload_size" db:"max_upload_size"`
}

func (p *FilePolicy) Patch(path *FilePolicyPath) {
	p.UpdatedBy = &path.UpdatedBy
	p.UpdatedAt = &path.UpdatedAt

	if path.Name != nil {
		p.Name = *path.Name
	}
	if path.Description != nil {
		p.Description = *path.Description
	}
	if path.Enabled != nil {
		p.Enabled = *path.Enabled
	}
	if path.MimeTypes != nil {
		p.MimeTypes = path.MimeTypes
	}
	if path.Channels != nil {
		p.Channels = path.Channels
	}
	if path.SpeedDownload != nil {
		p.SpeedDownload = *path.SpeedDownload
	}
	if path.SpeedUpload != nil {
		p.SpeedUpload = *path.SpeedUpload
	}
	if path.RetentionDays != nil {
		p.RetentionDays = *path.RetentionDays
	}
}

type SearchFilePolicy struct {
	ListRequest
	Ids []uint32
}

func (FilePolicy) DefaultOrder() string {
	return "position"
}

func (FilePolicy) AllowFields() []string {
	return []string{"id", "created_at", "created_by", "updated_at", "updated_by", "position", "max_upload_size",
		"name", "description", "enabled", "mime_types", "channels", "speed_download", "speed_upload", "retention_days",
	}
}

func (FilePolicy) DefaultFields() []string {
	return []string{"id", "position", "name", "description", "enabled", "channels", "mime_types"}
}

func (FilePolicy) EntityName() string {
	return "file_policies_view"
}

func (c *FilePolicy) IsValid() engine.AppError {

	return nil
}
