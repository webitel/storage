package app

import (
	"github.com/webitel/engine/auth_manager"
	engine "github.com/webitel/engine/model"
)

func (app *App) GetSession(token string) (*auth_manager.Session, engine.AppError) {
	session, err := app.sessionManager.GetSession(token)

	if err != nil {
		switch err {
		case auth_manager.ErrInternal:
			return nil, engine.NewInternalError("app.session.app_error", err.Error())

		case auth_manager.ErrStatusForbidden:
			return nil, engine.NewInternalError("app.session.forbidden", err.Error())

		case auth_manager.ErrValidId:
			return nil, engine.NewInternalError("app.session.is_valid.id.app_error", err.Error())

		case auth_manager.ErrValidUserId:
			return nil, engine.NewInternalError("app.session.is_valid.user_id.app_error", err.Error())

		case auth_manager.ErrValidToken:
			return nil, engine.NewInternalError("app.session.is_valid.token.app_error", err.Error())

		case auth_manager.ErrValidRoleIds:
			return nil, engine.NewInternalError("app.session.is_valid.role_ids.app_error", err.Error())
		default:
			return nil, engine.NewInternalError("app.session.unauthorized.app_error", err.Error())
		}
	}

	return session, nil
}
