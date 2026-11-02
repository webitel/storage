package model

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
)

const (
	BackendCacheSize               = 1000
	BackendProfileAccessKeyField   = "access_key"
	BackendProfilePathPatternField = "path_pattern"
)

type BackendProfileType string

const (
	FileDriverUnknown BackendProfileType = "unknown"
	FileDriverLocal   BackendProfileType = "local"
	FileDriverS3      BackendProfileType = "s3"
	FileDriverGDrive  BackendProfileType = "g_drive"
	FileDriverDropBox BackendProfileType = "drop_box"
)

type FileBackendProfileType struct {
	Id   int    `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
	Code string `db:"code" json:"code"`
}

type FileBackendProfile struct {
	DomainRecord
	Name        string             `db:"name" json:"name"`
	Description string             `json:"description" db:"description"`
	ExpireDay   int                `db:"expire_days" json:"expire_days"`
	Priority    int                `db:"priority" json:"priority"`
	Disabled    bool               `db:"disabled" json:"disabled"`
	MaxSizeMb   int                `db:"max_size" json:"max_size"`
	Properties  StringInterface    `db:"properties" json:"properties"`
	Type        BackendProfileType `db:"type" json:"type"`
	DataSize    float64            `db:"data_size" json:"data_size"`
	DataCount   int64              `db:"data_count" json:"data_count"`
}

type SearchFileBackendProfile struct {
	ListRequest
	Ids []uint32
}

func (FileBackendProfile) DefaultOrder() string {
	return "id"
}

func (a FileBackendProfile) AllowFields() []string {
	return []string{"id", "name", "max_size", "type", "expire_days", "disabled",
		"description", "priority", "properties", "data_size", "data_count",
		"domain_id", "created_at", "created_by", "updated_at", "updated_by"}
}

func (a FileBackendProfile) DefaultFields() []string {
	return []string{"id", "name", "max_size", "type", "expire_days", "disabled"}
}

func (a FileBackendProfile) EntityName() string {
	return "file_backend_profiles_view"
}

type S3Properties struct {
	KeyId      string   `json:"key_id"`
	AccessKey  string   `json:"access_key"`
	BucketName string   `json:"bucket_name"`
	Region     S3Region `json:"region"`
}

type DOProperties struct {
	KeyId      string   `json:"key_id"`
	AccessKey  string   `json:"access_key"`
	BucketName string   `json:"bucket_name"`
	Region     DORegion `json:"region"`
}

type DropBoxProperties struct {
	Token string `json:"token"`
}

type GDriveProperties struct {
	Email      string `json:"email"`
	PrivateKey string `json:"private_key"`
	Directory  string `json:"directory"`
}

type S3Region string
type DORegion string

const (
	S3UsEast1      S3Region = "us-east-1"
	S3UsWest1               = "us-west-1"
	S3UsWest2               = "us-west-2"
	S3ApSouth1              = "ap-south-1"
	S3ApNorthEast2          = "ap-northeast-2"
	S3ApSouthEast1          = "ap-southeast-1"
	S3ApSouthEast2          = "ap-southeast-2"
	S3NorthEast1            = "ap-northeast-1"
	S3EuCentral1            = "eu-central-1"
	S3EuWest1               = "eu-west-1"
	S3SaEast1               = "sa-east-1"
)

const (
	DONyc3 DORegion = "nyc3"
)

type FileBackendProfileSync struct {
	UpdatedAt int64 `json:"updated_at" db:"updated_at"`
	Disabled  bool  `json:"disabled" db:"disabled"`
}

type DomainFileBackendHashKey struct {
	Id        int   `db:"id"`
	UpdatedAt int64 `db:"updated_at"`
}

type FileBackendProfilePath struct {
	Name        *string          `json:"name"`
	ExpireDay   *int             `json:"expire_day"`
	Priority    *int             `json:"priority"`
	Disabled    *bool            `json:"disabled"`
	MaxSizeMb   *int             `json:"max_size_mb"`
	Properties  *StringInterface `json:"properties"`
	Description *string          `json:"description"`
	UpdatedBy   *Lookup
	UpdatedAt   int64
}

func (t BackendProfileType) String() string {
	return string(t)
}

func (p *FileBackendProfile) GetJsonProperties() string {
	d, _ := json.Marshal(p.Properties)
	return string(d)
}

func (f *FileBackendProfile) PreSave() {
	f.CreatedAt = GetMillis()
	f.UpdatedAt = f.CreatedAt
}

func (f *FileBackendProfile) IsValid() AppError {
	// on create action access_key from properties can't be empty
	// on update action access_key can be empty (task: WTEL-4344)
	if len(f.Name) == 0 {
		return NewBadRequestError("model.file_backend_profile.name.app_error", "")
	}

	//FIXME
	//if f.TypeId != 1 {
	//	return NewBadRequestError("model.file_backend_profile.type_id.app_error", "")
	//}
	return nil
}

// storePathPatternVars are the placeholders a path pattern is made of; see
// parseStorePattern in utils.
var storePathPatternVars = []string{"$DOMAIN", "$Y", "$M", "$D", "$H", "$m", "$CHANNEL", "$MIME"}

// IsValidPathPattern checks the path pattern set by a create or an update,
// so the pattern cannot be a path of its own ("../2/$Y", "/etc") that puts
// files outside the domain's tree. A pattern is placeholders joined by "/"
// (empty is no subdirectory). Profiles stored before this check may have
// any pattern: old is the stored profile on an update (nil on a create),
// and a pattern it already has is not checked again.
func (f *FileBackendProfile) IsValidPathPattern(old *FileBackendProfile) AppError {
	pattern := f.Properties.GetString(BackendProfilePathPatternField)
	if old != nil && old.Properties.GetString(BackendProfilePathPatternField) == pattern {
		return nil
	}
	if pattern == "" {
		return nil
	}
	for _, el := range strings.Split(pattern, "/") {
		if !slices.Contains(storePathPatternVars, el) {
			return NewBadRequestError("model.file_backend_profile.path_pattern.app_error",
				fmt.Sprintf("path pattern %q: %q is not one of %s joined by \"/\"", pattern, el, strings.Join(storePathPatternVars, ", ")))
		}
	}
	return nil
}

func (f *FileBackendProfile) ToJson() string {
	b, _ := json.Marshal(f)
	return string(b)
}

func (f *FileBackendProfile) Patch(path *FileBackendProfilePath) {
	f.UpdatedBy = path.UpdatedBy
	f.UpdatedAt = path.UpdatedAt

	if path.Name != nil {
		f.Name = *path.Name
	}

	if path.ExpireDay != nil {
		f.ExpireDay = *path.ExpireDay
	}

	if path.Priority != nil {
		f.Priority = *path.Priority
	}

	if path.Disabled != nil {
		f.Disabled = *path.Disabled
	}

	if path.MaxSizeMb != nil {
		f.MaxSizeMb = *path.MaxSizeMb
	}

	if path.Properties != nil {
		props := *path.Properties
		// an empty access_key keeps the stored one (WTEL-4344)
		if props.GetString(BackendProfileAccessKeyField) == "" {
			if old := f.Properties.GetString(BackendProfileAccessKeyField); old != "" {
				if props == nil {
					props = StringInterface{}
				}
				props[BackendProfileAccessKeyField] = old
			}
		}
		f.Properties = props
	}

	if path.Description != nil {
		f.Description = *path.Description
	}
}

func FileBackendProfileFromJson(data io.Reader) *FileBackendProfile {
	var profile FileBackendProfile
	if err := json.NewDecoder(data).Decode(&profile); err == nil {
		return &profile
	} else {
		return nil
	}
}

func FileBackendProfilePathFromJson(data io.Reader) *FileBackendProfilePath {
	var profile FileBackendProfilePath
	if err := json.NewDecoder(data).Decode(&profile); err == nil {
		return &profile
	} else {
		return nil
	}
}

func FileBackendProfileListToJson(list []*FileBackendProfile) string {
	b, _ := json.Marshal(list)
	return string(b)
}

func StorageBackendTypeFromString(t string) BackendProfileType {
	switch t {
	case FileDriverLocal.String():
		return FileDriverLocal

	case FileDriverS3.String():
		return FileDriverS3

	case FileDriverGDrive.String():
		return FileDriverGDrive

	case FileDriverDropBox.String():
		return FileDriverDropBox
	default:
		return FileDriverUnknown

	}
}
