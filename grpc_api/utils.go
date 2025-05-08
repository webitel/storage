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
