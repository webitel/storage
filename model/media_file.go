package model

import (
	"encoding/json"

	engine "github.com/webitel/engine/model"
)

type MediaFile struct {
	BaseFile
	DomainRecord
	DomainName string `json:"-" db:"domain_name"`
}

type SearchMediaFile struct {
	ListRequest
	Ids []uint32
}

func (MediaFile) DefaultOrder() string {
	return "id"
}

func (a MediaFile) AllowFields() []string {
	return []string{"id", "name", "mime_type", "size", "domain_id", "created_at", "created_by", "updated_at", "updated_by"}
}

func (a MediaFile) DefaultFields() []string {
	return []string{"id", "name", "mime_type", "size", "created_at"}
}

func (a MediaFile) EntityName() string {
	return "media_files_view"
}

func (self *MediaFile) PreSave() engine.AppError {
	self.CreatedAt = GetMillis()
	self.UpdatedAt = self.CreatedAt
	return nil
}

func (f *MediaFile) IsValid() engine.AppError {
	if len(f.Name) < 3 {
		return engine.NewBadRequestError("model.media.is_valid.name.app_error", "name="+f.Name)
	}

	if len(f.MimeType) < 3 {
		return engine.NewBadRequestError("model.media.is_valid.mime_type.app_error", "name="+f.Name)
	}

	if f.DomainId == 0 {
		return engine.NewBadRequestError("model.media.is_valid.domain_id.app_error", "name="+f.Name)
	}

	if f.Size == 0 {
		//FIXME
		//return NewBadRequestError("model.media.is_valid.size.app_error", "name="+f.Name)
	}
	return nil
}

func (self MediaFile) GetStoreName() string {
	return self.Name
}

func (self MediaFile) EncryptedKey() *string {
	return nil
}

func (self *MediaFile) ToJson() string {
	b, _ := json.Marshal(self)
	return string(b)
}

func (self *MediaFile) Domain() int64 {
	return self.DomainId
}
