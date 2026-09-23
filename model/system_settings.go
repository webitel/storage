package model

import (
	"fmt"
	"strconv"
	"strings"
)

const SystemSettingsObjectName = "system_settings"

type SystemSettingEvent struct {
	Name     string
	DomainID int64
}

func NewSystemSettingEventFromRoutingKey(rk string) (*SystemSettingEvent, AppError) {
	parts := strings.Split(rk, ".")
	if len(parts) < 4 {
		return nil, NewBadRequestError(
			"model.system_settings.new_system_setting_event.invalid_rk_len",
			fmt.Sprintf("received routing key %q with len less than 4", rk),
		)
	}

	if parts[0] != SystemSettingsObjectName || parts[1] == "" {
		return nil, NewBadRequestError(
			"model.system_settings.new_system_setting_event.invalid_rk",
			fmt.Sprintf("received unexpected routing key %q", rk),
		)
	}

	domainID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil || domainID <= 0 {
		return nil, NewBadRequestError(
			"model.system_settings.new_system_setting_event.invalid_domain_id",
			fmt.Sprintf("received invalid domain id in routing key %q", rk),
		)
	}

	return &SystemSettingEvent{Name: parts[1], DomainID: domainID}, nil
}

func SystemSettingCacheKey(domainID int64, name string) string {
	return fmt.Sprintf("%d-%s", domainID, name)
}
