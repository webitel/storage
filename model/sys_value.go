package model

import (
	"encoding/json"
	"strconv"
)

type SysValue json.RawMessage

func (v *SysValue) Int() *int {
	if v == nil {
		return nil
	}

	i, err := strconv.Atoi(string(*v))
	if err != nil {
		return nil
	}

	return &i
}
