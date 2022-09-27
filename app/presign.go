package app

import (
	"fmt"
	"net/http"

	"github.com/webitel/storage/model"
)

func (a *App) ValidateSignature(plain, signature string) bool {
	return a.preSigned.Valid(plain, signature)
}

func (a *App) GenerateSignature(msg []byte) (string, *model.AppError) {
	signature, err := a.preSigned.Generate(msg)
	if err != nil {
		return "", model.NewAppError("GenerateSignature", "app.signature.generate.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return signature, nil
}

func (a *App) GeneratePreSignetResourceSignature(resource, action string, id int64, domainId int64) (string, *model.AppError) {
	key := fmt.Sprintf("%s/%d/%s?domain_id=%d&expires=%d", resource, id, action, domainId,
		model.GetMillis()+a.Config().PreSignedTimeout)

	signature, err := a.GenerateSignature([]byte(key))
	if err != nil {
		return "", err
	}

	return key + "&signature=" + signature, nil

}
