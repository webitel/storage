package model

import "time"

type ImportTemplate struct {
	Id          int32           `json:"id" db:"id"`
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description" db:"description"`
	SourceType  string          `json:"source_type" db:"source_type"`
	SourceId    int64           `json:"source_id" db:"source_id"`
	Parameters  StringInterface `json:"parameters" db:"parameters"`
	Source      *Lookup         `json:"source" db:"source"`
	CreatedAt   *time.Time      `json:"created_at" db:"created_at"`
	CreatedBy   *Lookup         `json:"created_by" db:"created_by"`
	UpdatedAt   *time.Time      `json:"updated_at" db:"updated_at"`
	UpdatedBy   *Lookup         `json:"updated_by" db:"updated_by"`
}

type ImportTemplatePatch struct {
	Name        *string                `json:"name" db:"name"`
	Description *string                `json:"description" db:"description"`
	SourceType  *string                `json:"source_type" db:"source_type"`
	SourceId    *int64                 `json:"source_id" db:"source_id"`
	Parameters  map[string]interface{} `json:"parameters" db:"parameters"`
	Source      *Lookup                `json:"source" db:"source"`

	UpdatedBy Lookup
	UpdatedAt time.Time
}

type SearchImportTemplate struct {
	ListRequest
	Ids []int32
}

func (ImportTemplate) DefaultOrder() string {
	return "id"
}

func (ImportTemplate) AllowFields() []string {
	return []string{"id", "name", "description", "source_type", "source_id", "parameters", "source", "updated_at",
		"updated_by", "created_at", "created_at"}
}

func (ImportTemplate) DefaultFields() []string {
	return []string{"id", "name", "description", "source_type", "source_id", "parameters", "source"}
}

func (ImportTemplate) EntityName() string {
	return "import_template_view"
}

func (i *ImportTemplate) IsValid() *AppError {
	return nil
}

func (i *ImportTemplate) Path(path *ImportTemplatePatch) {
	i.UpdatedBy = &path.UpdatedBy
	i.UpdatedAt = &path.UpdatedAt

	if path.Name != nil {
		i.Name = *path.Name
	}
	if path.Description != nil {
		i.Description = *path.Description
	}
	if path.SourceType != nil {
		i.SourceType = *path.SourceType
	}
	if path.SourceId != nil {
		i.SourceId = *path.SourceId
	}
	if path.Parameters != nil {
		i.Parameters = path.Parameters
	}
}
