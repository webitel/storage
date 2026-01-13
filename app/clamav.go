package app

import (
	"fmt"
	"github.com/webitel/storage/app/clamd"
	"github.com/webitel/storage/model"
	"github.com/webitel/wlog"
)

const (
	ClamavModeAggressive = "aggressive"
	ClamavModeQuarantine = "quarantine"
	ClamavModeSkip       = "skip"
)

type Clamav struct {
	*clamd.Clamd
	mode string
}

func NewClamav(cfg model.ClamavSettings) *Clamav {
	c := &Clamav{mode: cfg.Mode}
	c.Clamd = clamd.NewClamd(cfg.Address)
	e := c.Clamd.Ping()
	if e != nil {
		wlog.Warn(fmt.Sprintf("clamav ping error: %s", e))
	}

	return c
}

func (c *Clamav) IsStoreFile(fv *model.MalwareScan) bool {
	if c.mode == ClamavModeAggressive {
		return false
	} else if c.mode == ClamavModeQuarantine {
		fv.Quarantine = true
	}

	return true
}
