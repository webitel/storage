package model_test

import (
	"testing"

	"github.com/webitel/storage/model"
)

func TestNewSystemSettingEventFromRoutingKey(t *testing.T) {
	tests := []struct {
		name    string
		rk      string
		want    *model.SystemSettingEvent
		wantErr bool
	}{
		{"update", "system_settings.period_to_playback_records.update.1.10", &model.SystemSettingEvent{Name: "period_to_playback_records", DomainID: 1}, false},
		{"delete", "system_settings.period_to_playback_records.delete.25.3", &model.SystemSettingEvent{Name: "period_to_playback_records", DomainID: 25}, false},
		{"without user", "system_settings.enable_2fa.create.7", &model.SystemSettingEvent{Name: "enable_2fa", DomainID: 7}, false},
		{"too short", "system_settings.enable_2fa.update", nil, true},
		{"wrong object", "domains.enable_2fa.update.1.1", nil, true},
		{"empty name", "system_settings..update.1.1", nil, true},
		{"domain not a number", "system_settings.enable_2fa.update.abc.1", nil, true},
		{"domain zero", "system_settings.enable_2fa.update.0.1", nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := model.NewSystemSettingEventFromRoutingKey(tc.rk)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if *got != *tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestSystemSettingCacheKey(t *testing.T) {
	if got := model.SystemSettingCacheKey(12, "period_to_playback_records"); got != "12-period_to_playback_records" {
		t.Fatalf("got %q", got)
	}
}
