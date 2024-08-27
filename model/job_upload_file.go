package model

import (
	"fmt"
)

const (
	UploadFileChannelCall = "call"
	UploadFileChannelChat = "chat"
	UploadFileChannelMail = "mail"
)

type JobUploadFile struct {
	Id        int64   `db:"id"`
	State     int     `db:"state"`
	Name      string  `db:"name"`
	ViewName  *string `db:"view_name"`
	Uuid      string  `db:"uuid"`
	DomainId  int64   `db:"domain_id"`
	MimeType  string  `db:"mime_type"`
	Size      int64   `db:"size"`
	EmailMsg  string  `db:"email_msg"`
	EmailSub  string  `db:"email_sub"`
	Instance  string  `db:"instance"`
	CreatedAt int64   `db:"created_at"`
	UpdatedAt int64   `db:"updated_at"`
	Attempts  int     `db:"attempts,default:0" json:"attempts"`
	SHA256Sum *string `db:"sha256sum" json:"sha256sum"`
	Channel   *string `db:"channel" json:"channel"`
}

type JobUploadFileWithProfile struct {
	JobUploadFile
	ProfileId        *int   `json:"profile_id" db:"profile_id"`
	ProfileUpdatedAt *int64 `json:"profile_updated_at" db:"profile_updated_at"`
}

func (self *JobUploadFile) PreSave() {
	if self.CreatedAt == 0 {
		self.CreatedAt = GetMillis()
	}
	self.UpdatedAt = GetMillis()
}

func (f *JobUploadFile) GetSize() int64 {
	return f.Size
}

func (f *JobUploadFile) GetMimeType() string {
	return f.MimeType
}

func (self *JobUploadFile) GetStoreName() string {
	return fmt.Sprintf("%s_%s", self.Uuid, self.Name)
}

func (self *JobUploadFile) GetViewName() string {
	if self.ViewName != nil {
		return *self.ViewName
	}

	return self.Name
}

// TODO
func (self *JobUploadFile) GetPropertyString(name string) string {
	return ""
}
func (self *JobUploadFile) SetPropertyString(name, value string) {

}
func (self *JobUploadFile) Domain() int64 {
	return self.DomainId
}
