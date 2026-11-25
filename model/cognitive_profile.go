package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	CognitiveProfileKeyField         = "key"
	CognitiveProfileRegionField      = "region"
	CognitiveProfileKeyLocationField = "key_location"

	CognitiveServiceTTS = "TTS"

	GoogleServiceAccountType = "service_account"
	GoogleTokenURI           = "https://oauth2.googleapis.com/token"

	maxCloudRegionLength = 64
)

var cloudRegionPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func IsValidCloudRegion(region string) bool {
	return len(region) <= maxCloudRegionLength && cloudRegionPattern.MatchString(region)
}

func ValidateGoogleServiceAccount(data []byte) error {
	var key struct {
		Type     string `json:"type"`
		TokenURI string `json:"token_uri"`
	}
	if err := json.Unmarshal(data, &key); err != nil {
		return errors.New("credentials must be a JSON object")
	}
	if key.Type != GoogleServiceAccountType {
		return fmt.Errorf("credentials type must be %q", GoogleServiceAccountType)
	}
	if key.TokenURI != "" && key.TokenURI != GoogleTokenURI {
		return fmt.Errorf("token_uri must be %q", GoogleTokenURI)
	}
	return nil
}

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

func (c *CognitiveProfile) IsValid() AppError {
	if _, ok := c.Properties[CognitiveProfileKeyLocationField]; ok {
		return NewBadRequestError("model.cognitive_profile.key_location.app_error", "key_location is not supported")
	}

	region := c.Properties.GetString(CognitiveProfileRegionField)
	if region != "" && !IsValidCloudRegion(region) {
		return NewBadRequestError("model.cognitive_profile.region.app_error", "invalid region")
	}

	switch strings.ToLower(c.Provider) {
	case "microsoft":
		if region == "" {
			return NewBadRequestError("model.cognitive_profile.region.app_error", "region is required")
		}
	case "google":
		if !strings.EqualFold(c.Service, CognitiveServiceTTS) {
			break
		}
		key, ok := c.Properties[CognitiveProfileKeyField]
		if !ok || key == nil || key == "" {
			break
		}
		data, err := json.Marshal(key)
		if err == nil {
			err = ValidateGoogleServiceAccount(data)
		}
		if err != nil {
			return NewBadRequestError("model.cognitive_profile.key.app_error", err.Error())
		}
	}

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
		if path.Properties.GetString(CognitiveProfileKeyField) == "" {
			if oldKey, ok := f.Properties[CognitiveProfileKeyField]; ok {
				(*path.Properties)[CognitiveProfileKeyField] = oldKey
			}
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
