package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/h2non/filetype"
	"github.com/juju/ratelimit"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
	"github.com/webitel/wlog"
	"golang.org/x/sync/singleflight"
	"io"
	"strings"
	"time"
)

var (
	ErrorMaxLimit      = errors.New("max size")
	ErrorExtUnknown    = errors.New("extension of file is unknown")
	ErrorExtSuspicious = errors.New("actual file extension doesn't match declared Content-Type")
	ErrorExtNotAllowed = errors.New("file extension is not allowed")
)

var (
	policiesStoreGroup singleflight.Group
)

type PolicyReader struct {
	r          io.ReadCloser // underlying reader
	bytesCount int64
	bucket     *ratelimit.Bucket
	f          *model.BaseFile
	maxSize    int64
	mimeTyme   string
}

type FilePolicy struct {
	mime []string

	speedDownload int64
	speedUpload   int64
	maxUploadSize int64
	retentionDays int
}

type PoliciesHub struct {
	app      *App
	id       int64
	policies []*FilePolicy
	channels map[string][]*FilePolicy
	log      *wlog.Logger
}

var (
	FilePolicyAllowAll = &FilePolicy{
		mime: []string{"*"},
	}
)

type DomainFilePolicy struct {
	app      *App
	policies utils.ObjectCache
}

func (app *App) FilePolicyForDownload(domainId int64, file *model.BaseFile, src io.ReadCloser) (io.ReadCloser, engine.AppError) {
	return app.filePolicies.policyReaderForDownload(domainId, file, src)
}

func (app *App) FilePolicyForUpload(domainId int64, file *model.BaseFile, src io.ReadCloser) (io.ReadCloser, engine.AppError) {
	return app.filePolicies.policyReaderForUpload(domainId, file, src)
}

func (app *App) policiesHub(domainId int64) (*PoliciesHub, engine.AppError) {
	policies, err := app.Store.FilePolicies().AllByDomainId(context.Background(), domainId)
	if err != nil {
		return nil, err
	}

	return app.newPoliciesHub(domainId, policies), nil
}

func (app *App) newPoliciesHub(domainId int64, policies []model.FilePolicy) *PoliciesHub {

	h := PoliciesHub{
		channels: make(map[string][]*FilePolicy),
		id:       domainId,
		log:      app.Log.With(wlog.Int64("hub_id", domainId)),
	}

	for _, v := range policies {
		if len(v.MimeTypes) == 0 {
			// skip
			continue
		}

		p := FilePolicy{
			speedDownload: v.SpeedDownload * 1024, // kbs
			speedUpload:   v.SpeedUpload * 1024,   // kbs
			maxUploadSize: v.MaxUploadSize,        // bytes
			mime:          v.MimeTypes,
			retentionDays: int(v.RetentionDays),
		}

		h.appendPolicy(v.Channels, &p)

	}

	return &h
}

func (app *App) cachedPolicyHub(domainId int64) (*PoliciesHub, engine.AppError) {
	var err error
	var shared bool
	h, ok := app.filePolicies.policies.Get(domainId)
	if ok {
		return h.(*PoliciesHub), nil
	}

	h, err, shared = policiesStoreGroup.Do(fmt.Sprintf("%d", domainId), func() (interface{}, error) {
		h, err := app.policiesHub(domainId)
		if err != nil {
			return nil, err
		}

		return h, nil
	})

	if err != nil {
		switch err.(type) {
		case engine.AppError:
			return nil, err.(engine.AppError)
		default:
			return nil, engine.NewInternalError("app.file_policies.cached", err.Error())
		}
	}

	if !shared {
		app.filePolicies.policies.AddWithDefaultExpires(domainId, h)
	}

	return h.(*PoliciesHub), nil
}

func (ph *DomainFilePolicy) policyReaderForDownload(domainId int64, file *model.BaseFile, src io.ReadCloser) (io.ReadCloser, engine.AppError) {
	var policy *FilePolicy
	v, err := ph.app.cachedPolicyHub(domainId)
	if err != nil {
		return nil, err
	}
	policy, err = v.Policy(file.Channel, file.MimeType)
	if err != nil {
		return nil, err
	}

	if policy == FilePolicyAllowAll {
		return src, nil
	}

	r := &PolicyReader{
		r:        src,
		f:        file,
		mimeTyme: file.MimeType,
	}

	if policy.speedDownload > 0 {
		r.bucket = ratelimit.NewBucketWithRate(float64(policy.speedDownload), policy.speedDownload)
	}

	return r, nil
}

func (ph *DomainFilePolicy) policyReaderForUpload(domainId int64, file *model.BaseFile, src io.ReadCloser) (io.ReadCloser, engine.AppError) {
	var policy *FilePolicy
	v, err := ph.app.cachedPolicyHub(domainId)
	if err != nil {
		return nil, err
	}
	policy, err = v.Policy(file.Channel, file.MimeType)
	if err != nil {
		return nil, err
	}

	if policy == FilePolicyAllowAll {
		return src, nil
	}

	r := &PolicyReader{
		r:       src,
		f:       file,
		maxSize: policy.maxUploadSize,
	}

	if file.Channel == nil || *file.Channel != model.UploadFileChannelMedia {
		// TODO check all channel ?
		r.mimeTyme = file.MimeType
	}

	if policy.retentionDays > 0 {
		t := time.Now().AddDate(0, 0, policy.retentionDays)
		file.RetentionUntil = &t
	}

	if policy.speedUpload > 0 {
		r.bucket = ratelimit.NewBucketWithRate(float64(policy.speedUpload), policy.speedUpload)
	}

	return r, nil
}

func (ph *PoliciesHub) appendPolicy(channels []string, policy *FilePolicy) {
	ph.policies = append(ph.policies, policy)
	for _, c := range channels {
		ch, _ := ph.channels[c]

		ch = append(ch, policy)
		ph.channels[c] = ch
	}
}

func (ph *PoliciesHub) Policy(channel *string, mime string) (*FilePolicy, engine.AppError) {
	if channel == nil {
		// TODO
		return nil, engine.NewNotFoundError("app.file_policy.valid.channel", "Not found")
	}
	policies, ok := ph.channels[*channel]
	if !ok {
		return FilePolicyAllowAll, nil
	}

	for _, policy := range policies {
		for _, m := range policy.mime {
			if MatchPattern(m, mime) {
				return policy, nil
			}
		}
	}

	return nil, engine.NewForbiddenError("app.file.policy", "Forbidden")
}

func (r *PolicyReader) Read(buf []byte) (n int, err error) {
	n, err = r.r.Read(buf)
	if n <= 0 {
		return
	}
	r.bytesCount += int64(n)

	if r.maxSize > 0 && r.bytesCount > r.maxSize {
		err = ErrorMaxLimit
		return
	}

	if r.mimeTyme == "" {
		err = r.testMimeType(buf)
		if err != nil {
			return n, err
		}
		r.mimeTyme = r.f.MimeType
	}

	if r.bucket != nil {
		r.bucket.Wait(int64(n))
	}

	return
}

func (r *PolicyReader) Close() (err error) {
	return r.r.Close()
}

func (r *PolicyReader) testMimeType(bytes []byte) error {
	kind, err := filetype.Match(bytes)
	if err != nil {
		return err
	}

	if kind == filetype.Unknown {
		// TODO
		return ErrorExtUnknown
	}

	if strings.HasPrefix(r.f.MimeType, kind.MIME.Value) || (r.f.MimeType == "audio/wav" && kind.Extension == "wav") {
		return nil
	}

	if r.f.MimeType != kind.MIME.Value {
		return ErrorExtSuspicious
	}

	// File mime type is not in the allowed list
	return ErrorExtNotAllowed
}

// MatchPattern перевіряє, чи відповідає рядок заданому патерну
func MatchPattern(pattern, str string) bool {
	pLen := len(pattern)
	sLen := len(str)
	pIdx, sIdx := 0, 0
	starIdx, matchIdx := -1, 0

	for sIdx < sLen {
		if pIdx < pLen && (pattern[pIdx] == '?' || pattern[pIdx] == str[sIdx]) {
			// Якщо символ патерну '?' або символи збігаються, рухаємося далі
			pIdx++
			sIdx++
		} else if pIdx < pLen && pattern[pIdx] == '*' {
			// Якщо символ патерну '*', зберігаємо позицію зірочки
			starIdx = pIdx
			matchIdx = sIdx
			pIdx++
		} else if starIdx != -1 {
			// Якщо є збережена зірочка, використовуємо її для пропуску символу
			pIdx = starIdx + 1
			matchIdx++
			sIdx = matchIdx
		} else {
			// Інакше рядок не відповідає патерну
			return false
		}
	}

	// Перевіряємо, чи залишилися невідповідні символи в патерні
	for pIdx < pLen && pattern[pIdx] == '*' {
		pIdx++
	}

	return pIdx == pLen
}
