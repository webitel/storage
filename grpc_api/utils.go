package grpc_api

import (
	"github.com/webitel/storage/gen/engine"
	"github.com/webitel/storage/model"
)

func GetProtoLookup(src *model.Lookup) *engine.Lookup {
	if src == nil {
		return nil
	}

	return &engine.Lookup{
		Id:   int64(src.Id),
		Name: src.Name,
	}
}

func GetFilterBetween(src *engine.FilterBetween) *model.FilterBetween {
	if src == nil {
		return nil
	}

	return &model.FilterBetween{
		From: src.GetFrom(),
		To:   src.GetTo(),
	}
}
