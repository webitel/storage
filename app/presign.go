package app

import (
	"fmt"

	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
)

func (a *App) ValidateSignature(plain, signature string) bool {
	return a.preSigned.Valid(plain, signature)
}

func (a *App) GenerateSignature(msg []byte) (string, engine.AppError) {
	signature, err := a.preSigned.Generate(msg)
	if err != nil {
		return "", engine.NewInternalError("app.signature.generate.app_err", err.Error())
	}

	return signature, nil
}

func (a *App) GeneratePreSignetResourceSignature(resource, action string, id int64, domainId int64) (string, engine.AppError) {
	key := fmt.Sprintf("%s/%d/%s?domain_id=%d&expires=%d", resource, id, action, domainId,
		model.GetMillis()+a.Config().PreSignedTimeout)

	signature, err := a.GenerateSignature([]byte(key))
	if err != nil {
		return "", err
	}

	return key + "&signature=" + signature, nil

}
