package controller

import (
	"context"
	"github.com/webitel/engine/pkg/wbt/auth_manager"
	"github.com/webitel/storage/model"
)

func (c *Controller) GetSessionFromCtx(ctx context.Context) (*auth_manager.Session, model.AppError) {
	return c.app.GetSessionFromCtx(ctx)
}
