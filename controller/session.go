package controller

import (
	"context"

	"github.com/webitel/engine/auth_manager"
	engine "github.com/webitel/engine/model"
)

func (ctrl *Controller) GetSessionFromCtx(ctx context.Context) (*auth_manager.Session, engine.AppError) {
	return ctrl.app.GetSessionFromCtx(ctx)
}
