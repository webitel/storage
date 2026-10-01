package app

import (
	"context"

	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
	"golang.org/x/sync/singleflight"
)

var (
	systemCache = utils.NewLruWithParams(500, "system_settings", 15, "")
	systemGroup = singleflight.Group{}
)

func (a *App) GetCachedSystemSetting(ctx context.Context, domainId int64, name string) (model.SysValue, model.AppError) {
	key := model.SystemSettingCacheKey(domainId, name)
	c, ok := systemCache.Get(key)
	if ok {
		return c.(model.SysValue), nil
	}

	v, err, share := systemGroup.Do(key, func() (interface{}, error) {
		res, err := a.Store.SystemSettings().ValueByName(ctx, domainId, name)
		if err != nil {
			return model.SysValue{}, err
		}
		return res, nil
	})

	if err != nil {
		switch err.(type) {
		case model.AppError:
			return model.SysValue{}, err.(model.AppError)
		default:
			return model.SysValue{}, model.NewInternalError("app.sys_settings.get", err.Error())
		}
	}

	if !share {
		systemCache.AddWithDefaultExpires(key, v.(model.SysValue))
	}

	return v.(model.SysValue), nil
}

func (a *App) InvalidateCachedSystemSetting(e *model.SystemSettingEvent) {
	key := model.SystemSettingCacheKey(e.DomainID, e.Name)
	systemCache.Remove(key)
	systemGroup.Forget(key)
}
