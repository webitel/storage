package app

import (
	"context"
	"github.com/webitel/engine/pkg/wbt/auth_manager"
	"github.com/webitel/storage/model"
)

func (app *App) GetSession(token string) (*auth_manager.Session, model.AppError) {
	session, err := app.sessionManager.GetSession(context.Background(), token)

	if err != nil {
		switch err {
		case auth_manager.ErrInternal:
			return nil, model.NewInternalError("app.session.app_error", err.Error())

		case auth_manager.ErrStatusForbidden:
			return nil, model.NewInternalError("app.session.forbidden", err.Error())

		case auth_manager.ErrValidId:
			return nil, model.NewInternalError("app.session.is_valid.id.app_error", err.Error())

		case auth_manager.ErrValidUserId:
			return nil, model.NewInternalError("app.session.is_valid.user_id.app_error", err.Error())

		case auth_manager.ErrValidToken:
			return nil, model.NewInternalError("app.session.is_valid.token.app_error", err.Error())

		case auth_manager.ErrValidRoleIds:
			return nil, model.NewInternalError("app.session.is_valid.role_ids.app_error", err.Error())
		default:
			return nil, model.NewInternalError("app.session.unauthorized.app_error", err.Error())
		}
	}

	return session, nil
}
