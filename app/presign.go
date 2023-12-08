package app

import (
	"fmt"
	"net/url"
	"strconv"

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

func (a *App) GeneratePreSignedResourceSignature(resource, action string, id int64, domainId int64) (string, engine.AppError) {
	key := fmt.Sprintf("%s/%d/%s?domain_id=%d&expires=%d", resource, id, action, domainId,
		model.GetMillis()+a.Config().PreSignedTimeout)

	signature, err := a.GenerateSignature([]byte(key))
	if err != nil {
		return "", err
	}

	return key + "&signature=" + signature, nil

}

func (a *App) GeneratePreSignedResourceSignatureBulk(id, domainId int64, resource, action, source string, queryParams map[string]string) (string, engine.AppError) {
	var expire int64
	if v, ok := queryParams["expire"]; ok {
		val, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return "", engine.NewBadRequestError("app.presigned.generate_pre_signed_signature_bulk.parse_expire.error", err.Error())
		}
		expire = val
	} else {
		expire = model.GetMillis() + a.Config().PreSignedTimeout
	}
	base := fmt.Sprintf("%s/%d/%s?source=%s&domain_id=%d&expires=%d", resource, id, action, source, domainId,
		expire)
	uri, err := url.Parse(base)
	if err != nil {
		return "", engine.NewBadRequestError("app.presigned.generate_pre_signed_signature_bulk.parse.error", err.Error())
	}
	existingParams := uri.Query()
	for key, val := range queryParams {
		existingParams.Add(key, val)
	}
	uri.RawQuery = existingParams.Encode()

	signature, appErr := a.GenerateSignature([]byte(uri.RawQuery))
	if appErr != nil {
		return "", appErr
	}
	existingParams.Add("signature", signature)
	uri.RawQuery = existingParams.Encode()

	return uri.String(), nil

}
