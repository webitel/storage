package controller

import (
	"github.com/webitel/engine/auth_manager"
	"github.com/webitel/storage/model"
)

func (c *Controller) CreateImportTemplate(session *auth_manager.Session, template *model.ImportTemplate) (*model.ImportTemplate, *model.AppError) {
	var err *model.AppError

	if err = template.IsValid(); err != nil {
		return nil, err
	}

	return c.app.CreateImportTemplate(session.Domain(0), template)
}

func (c *Controller) SearchImportTemplates(session *auth_manager.Session, search *model.SearchImportTemplate) ([]*model.ImportTemplate, bool, *model.AppError) {
	var list []*model.ImportTemplate
	var err *model.AppError
	var endOfList bool

	list, endOfList, err = c.app.SearchImportTemplates(session.Domain(0), search)

	return list, endOfList, err
}

func (c *Controller) GetImportTemplate(session *auth_manager.Session, id int32) (*model.ImportTemplate, *model.AppError) {
	return c.app.GetImportTemplate(session.Domain(0), id)
}

func (c *Controller) UpdateImportTemplate(session *auth_manager.Session, template *model.ImportTemplate) (*model.ImportTemplate, *model.AppError) {
	var err *model.AppError

	if err = template.IsValid(); err != nil {
		return nil, err
	}

	return c.app.UpdateImportTemplate(session.Domain(0), template)
}

func (c *Controller) PatchImportTemplate(session *auth_manager.Session, id int32, patch *model.ImportTemplatePatch) (*model.ImportTemplate, *model.AppError) {
	return c.app.PatchImportTemplate(session.Domain(0), id, patch)
}

func (c *Controller) DeleteImportTemplate(session *auth_manager.Session, id int32) (*model.ImportTemplate, *model.AppError) {
	return c.app.DeleteImportTemplate(session.Domain(0), id)
}
