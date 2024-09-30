package grpc_api

import (
	engine "buf.build/gen/go/webitel/engine/protocolbuffers/go"
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
