package app

import (
	"strings"

	"github.com/webitel/engine/auth_manager"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
	tts2 "github.com/webitel/storage/tts"
)

type ttsVoiceFunction func(domainId int64, params *model.SearchCognitiveProfileVoice) ([]*model.CognitiveProfileVoice, engine.AppError)

var (
	ttsVoiceEngine = map[string]ttsVoiceFunction{
		strings.ToLower(TtsMicrosoft):  tts2.MicrosoftVoice,
		strings.ToLower(TtsGoogle):     tts2.GoogleVoice,
		strings.ToLower(TtsElevenLabs): tts2.ElevenLabsVoice,
	}
)

func (app *App) CognitiveProfileCheckAccess(domainId, id int64, groups []int, access auth_manager.PermissionAccess) (bool, engine.AppError) {
	return app.Store.CognitiveProfile().CheckAccess(domainId, id, groups, access)
}

func (app *App) CreateCognitiveProfile(profile *model.CognitiveProfile) (*model.CognitiveProfile, engine.AppError) {
	return app.Store.CognitiveProfile().Create(profile)
}

func (app *App) SearchCognitiveProfiles(domainId int64, search *model.SearchCognitiveProfile) ([]*model.CognitiveProfile, bool, engine.AppError) {
	res, err := app.Store.CognitiveProfile().GetAllPage(domainId, search)
	if err != nil {
		return nil, false, err
	}
	search.RemoveLastElemIfNeed(&res)
	return res, search.EndOfList(), nil
}

func (app *App) SearchCognitiveProfilesByGroups(domainId int64, groups []int, search *model.SearchCognitiveProfile) ([]*model.CognitiveProfile, bool, engine.AppError) {
	res, err := app.Store.CognitiveProfile().GetAllPageByGroups(domainId, groups, search)
	if err != nil {
		return nil, false, err
	}
	search.RemoveLastElemIfNeed(&res)
	return res, search.EndOfList(), nil
}

func (app *App) SearchCognitiveProfileVoices(domainId int64, search *model.SearchCognitiveProfileVoice) ([]*model.CognitiveProfileVoice, engine.AppError) {
	var ttsProfile *model.TtsProfile
	ttsProfile, err := app.Store.CognitiveProfile().SearchTtsProfile(domainId, int(search.Id))
	if err != nil {
		return nil, err
	}
	if !ttsProfile.Enabled {
		err = engine.NewBadRequestError("tts.profile.disabled", "Profile is disabled")

		return nil, err
	}

	provider := ttsProfile.Provider
	provider = strings.ToLower(provider)

	if fn, ok := ttsVoiceEngine[provider]; ok {
		res, ttsErr := fn(domainId, search)
		if ttsErr != nil {
			switch ttsErr.(type) {
			case engine.AppError:
				return nil, engine.NewNotFoundError("tts.valid.not_found", "Not found provider")
			default:
				return nil, engine.NewInternalError("tts.app_error", ttsErr.Error())
			}
		}
		return res, nil
	}

	return nil, engine.NewNotFoundError("tts.valid.not_found", "Not found provider")
}

func (app *App) GetCognitiveProfile(id, domain int64) (*model.CognitiveProfile, engine.AppError) {
	return app.Store.CognitiveProfile().Get(id, domain)
}

func (app *App) UpdateCognitiveProfile(profile *model.CognitiveProfile) (*model.CognitiveProfile, engine.AppError) {
	oldProfile, err := app.GetCognitiveProfile(profile.Id, profile.DomainId)
	if err != nil {
		return nil, err
	}

	oldProfile.UpdatedBy = profile.UpdatedBy
	oldProfile.UpdatedAt = profile.UpdatedAt

	oldProfile.Provider = profile.Provider
	// if access_key of profile is empty do not let reset access key (task: WTEL-4344)
	if oldAccessKey, newAccessKey := oldProfile.Properties.GetString(model.CognitiveProfileKeyField), profile.Properties.GetString(model.CognitiveProfileKeyField); newAccessKey == "" {
		profile.Properties[model.CognitiveProfileKeyField] = oldAccessKey
	}
	oldProfile.Properties = profile.Properties
	oldProfile.Enabled = profile.Enabled
	oldProfile.Name = profile.Name
	oldProfile.Description = profile.Description
	oldProfile.Service = profile.Service
	oldProfile.Default = profile.Default

	return app.Store.CognitiveProfile().Update(oldProfile)

}

func (app *App) PatchCognitiveProfile(domainId, id int64, patch *model.CognitiveProfilePath) (*model.CognitiveProfile, engine.AppError) {
	oldProfile, err := app.GetCognitiveProfile(id, domainId)
	if err != nil {
		return nil, err
	}

	oldProfile.Patch(patch)

	if err = oldProfile.IsValid(); err != nil {
		return nil, err
	}

	return app.Store.CognitiveProfile().Update(oldProfile)
}

func (app *App) DeleteCognitiveProfile(domainId, id int64) (*model.CognitiveProfile, engine.AppError) {
	profile, err := app.GetCognitiveProfile(id, domainId)
	if err != nil {
		return nil, err
	}
	err = app.Store.CognitiveProfile().Delete(domainId, id)
	if err != nil {
		return nil, err
	}

	return profile, nil
}
