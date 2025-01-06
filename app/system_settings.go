package app

import (
	"context"
	"fmt"
	"github.com/webitel/engine/model"
	"github.com/webitel/storage/utils"
	"golang.org/x/sync/singleflight"
)

var (
	systemCache = utils.NewLruWithParams(500, "system_settings", 15, "")
	systemGroup = singleflight.Group{}
)

func (a *App) GetCachedSystemSetting(ctx context.Context, domainId int64, name string) (model.SysValue, model.AppError) {
	key := fmt.Sprintf("%d-%s", domainId, name)
	c, ok := systemCache.Get(key)
	if ok {
		return c.(model.SysValue), nil
	}

	v, err, share := systemGroup.Do(fmt.Sprintf("%d-%s", domainId, name), func() (interface{}, error) {
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
