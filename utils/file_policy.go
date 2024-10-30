package utils

import (
	"github.com/webitel/storage/model"
	"github.com/webitel/wlog"
	"io"
	"regexp"
)

type PolicyReader struct {
	r        io.Reader // underlying reader
	n        int64
	f        *model.File
	speed    int64
	maxSize  int64
	mimeTyme string
}

type Policy struct {
	mime          *regexp.Regexp
	channel       string
	speedDownload int64
	speedUpload   int64
}

type PoliciesHub struct {
	policies []*Policy
	log      *wlog.Logger
}

func NewPoliciesHub(domainId int64, policies []model.FilePolicy) *PoliciesHub {
	h := PoliciesHub{
		policies: make([]*Policy, len(policies), len(policies)),
	}

	for _, v := range policies {
		if v.Enabled {

		}
	}

	return &h
}

type DomainFilePolicy struct {
	id       int64
	policies Map[int64, *PoliciesHub]
}

func (fp *DomainFilePolicy) PolicyReader(file *model.File, src io.Reader) (*PolicyReader, error) {
	panic(1)
}
