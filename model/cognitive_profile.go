package model

import (
	"encoding/json"
	"fmt"
	"time"

	engine "github.com/webitel/engine/model"
)

const CognitiveProfileKeyField = "key"

type CognitiveProfile struct {
	Id        int64      `json:"id" db:"id"`
	DomainId  int64      `json:"-" db:"domain_id"`
	CreatedAt *time.Time `json:"created_at" db:"created_at"`
	CreatedBy *Lookup    `json:"created_by" db:"created_by"`
	UpdatedAt *time.Time `json:"updated_at" db:"updated_at"`
	UpdatedBy *Lookup    `json:"updated_by" db:"updated_by"`

	Provider    string          `json:"provider" db:"provider"`
	Properties  StringInterface `json:"properties" db:"properties"`
	Enabled     bool            `json:"enabled" db:"enabled"`
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description" db:"description"`
	Service     string          `json:"service" db:"service"`
	Default     bool            `json:"default" db:"default"`
	Instance    interface{}     `json:"-" db:"-"`
	SyncTag     int64           `json:"-" db:"-"`
}

type CognitiveProfileVoice struct {
	Id   string `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type SearchCognitiveProfileVoice struct {
	ListRequest
	Id  int64
	Key string
}

type SearchCognitiveProfile struct {
	ListRequest
	Ids     []int64
	Enabled bool
	Service []string
}

func (c *CognitiveProfile) GetLocale(in *string) string {
	if in != nil {
		return *in
	}
	if v, ok := c.Properties["locale"]; ok {
		return fmt.Sprintf("%v", v)
	}

	return "en-US"
}

func (CognitiveProfile) DefaultOrder() string {
	return "id"
}

func (CognitiveProfile) AllowFields() []string {
	return []string{"id", "domain_id", "created_at", "created_by", "updated_at", "updated_by",
		"provider", "properties", "enabled", "name", "description", "service", "default",
	}
}

func (CognitiveProfile) DefaultFields() []string {
	return []string{"id", "provider", "properties", "enabled", "name", "description", "service", "default"}
}

func (CognitiveProfile) EntityName() string {
	return "cognitive_profile_services_view"
}

func (c *CognitiveProfile) IsValid() engine.AppError {
	// on create action key from properties can't be empty
	// on update action key can be empty (task: WTEL-4344)
	return nil
}

type CognitiveProfilePath struct {
	Provider    *string          `json:"provider" db:"provider"`
	Properties  *StringInterface `json:"properties" db:"properties"`
	Enabled     *bool            `json:"enabled" db:"enabled"`
	Name        *string          `json:"name" db:"name"`
	Description *string          `json:"description" db:"description"`
	Service     *string          `json:"service" db:"service"`
	Default     *bool            `json:"default" db:"default"`

	UpdatedBy Lookup
	UpdatedAt time.Time
}

func (f *CognitiveProfile) Patch(path *CognitiveProfilePath) {
	f.UpdatedBy = &path.UpdatedBy
	f.UpdatedAt = &path.UpdatedAt

	if path.Provider != nil {
		f.Provider = *path.Provider
	}

	if path.Properties != nil {
		if oldAccessKey, newAccessKey := f.Properties.GetString(CognitiveProfileKeyField), path.Properties.GetString(CognitiveProfileKeyField); newAccessKey == "" {
			f.Properties[CognitiveProfileKeyField] = oldAccessKey
		}
		f.Properties = *path.Properties
	}

	if path.Enabled != nil {
		f.Enabled = *path.Enabled
	}

	if path.Name != nil {
		f.Name = *path.Name
	}

	if path.Description != nil {
		f.Description = *path.Description
	}

	if path.Service != nil {
		f.Service = *path.Service
	}

	if path.Default != nil {
		f.Default = *path.Default
	}
}

func (c *CognitiveProfile) GetSyncTag() int64 {
	return c.SyncTag
}

func (c *CognitiveProfile) JsonProperties() []byte {
	data, _ := json.Marshal(c.Properties)
	return data
}
