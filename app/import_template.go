package app

import "github.com/webitel/storage/model"

func (app *App) CreateImportTemplate(domainId int64, template *model.ImportTemplate) (*model.ImportTemplate, *model.AppError) {
	return app.Store.ImportTemplate().Create(domainId, template)
}

func (app *App) SearchImportTemplates(domainId int64, search *model.SearchImportTemplate) ([]*model.ImportTemplate, bool, *model.AppError) {
	res, err := app.Store.ImportTemplate().GetAllPage(domainId, search)
	if err != nil {
		return nil, false, err
	}
	search.RemoveLastElemIfNeed(&res)
	return res, search.EndOfList(), nil
}

func (app *App) GetImportTemplate(domain int64, id int32) (*model.ImportTemplate, *model.AppError) {
	return app.Store.ImportTemplate().Get(domain, id)
}

func (app *App) UpdateImportTemplate(domainId int64, template *model.ImportTemplate) (*model.ImportTemplate, *model.AppError) {
	oldTemplate, err := app.GetImportTemplate(domainId, template.Id)
	if err != nil {
		return nil, err
	}

	oldTemplate.Name = template.Name
	oldTemplate.Description = template.Description
	oldTemplate.Parameters = template.Parameters
	oldTemplate.Source = template.Source
	oldTemplate.SourceId = template.SourceId
	oldTemplate.SourceType = template.SourceType
	if template.Source != nil {
		oldTemplate.SourceId = int64(template.Source.Id)
	}

	return app.Store.ImportTemplate().Update(domainId, oldTemplate)
}

func (app *App) PatchImportTemplate(domainId int64, id int32, patch *model.ImportTemplatePatch) (*model.ImportTemplate, *model.AppError) {
	oldTemplate, err := app.GetImportTemplate(domainId, id)
	if err != nil {
		return nil, err
	}

	oldTemplate.Path(patch)

	if err = oldTemplate.IsValid(); err != nil {
		return nil, err
	}

	return app.Store.ImportTemplate().Update(domainId, oldTemplate)
}

func (app *App) DeleteImportTemplate(domainId int64, id int32) (*model.ImportTemplate, *model.AppError) {
	template, err := app.GetImportTemplate(domainId, id)
	if err != nil {
		return nil, err
	}
	err = app.Store.ImportTemplate().Delete(domainId, id)
	if err != nil {
		return nil, err
	}

	return template, nil
}
