package utils

import (
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/webitel/storage/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/webitel/wlog"
)

const (
	SelCDN        = "selcdn.ru"
	GoogleStorage = "storage.googleapis.com"

	// Not ErrFileWriteExistsId: callers take that one as success and save a
	// file row pointing at the object that is already there.
	ErrObjectExistsId = "utils.file.s3.exists"
)

type S3FileBackend struct {
	BaseFileBackend
	name           string
	region         string
	accessKey      string
	accessToken    string
	bucket         string
	endpoint       string
	pathPattern    string
	sess           *session.Session
	svc            *s3.S3
	uploader       *s3manager.Uploader
	forcePathStyle bool
}

// objectKey joins parts into an object key and refuses one with a ".."
// element. Unlike a file system, the bucket has no root to escape from, but
// path.Join would still fold "<uuid>_<id>_/../../../etc/x" out of the
// directory prefix and let a client overwrite any object in the bucket. The
// directory itself is checked too: $MIME and $CHANNEL come from the request.
func objectKey(parts ...string) (string, model.AppError) {
	for _, p := range parts {
		for _, el := range strings.Split(p, "/") {
			if el == ".." {
				return "", model.NewBadRequestError(ErrPathEscapeId, fmt.Sprintf("object key %q leaves its directory", strings.Join(parts, "/")))
			}
		}
	}
	return path.Join(parts...), nil
}

func (self *S3FileBackend) Name() string {
	return self.name
}

func (self *S3FileBackend) GetStoreDirectory(f File) string {
	return path.Join(parseStorePattern(self.pathPattern, f))
}

func (self *S3FileBackend) getEndpoint() *string {
	if self.endpoint == "amazonaws.com" {
		return nil
	} else if self.region != "" && !isS3ForcePathStyle(self.endpoint) && !self.forcePathStyle {
		return aws.String(fmt.Sprintf("%s.%s", self.region, self.endpoint))
	} else {
		return aws.String(fmt.Sprintf("%s", self.endpoint))
	}
}

func isS3ForcePathStyle(name string) bool {
	return name == GoogleStorage || strings.HasSuffix(name, SelCDN)
}

func (self *S3FileBackend) TestConnection() model.AppError {
	config := &aws.Config{
		Region:      aws.String(strings.ToLower(self.region)),
		Endpoint:    self.getEndpoint(),
		Credentials: credentials.NewStaticCredentials(self.accessKey, self.accessToken, ""),
	}

	if isS3ForcePathStyle(self.endpoint) || self.forcePathStyle {
		config.S3ForcePathStyle = aws.Bool(true)
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return model.NewInternalError("utils.file.s3.test_connection.app_error", err.Error())
	}

	self.sess = sess
	self.svc = s3.New(sess)
	self.uploader = s3manager.NewUploader(sess)

	return nil
}

func (self *S3FileBackend) Write(src io.Reader, file File) (int64, model.AppError) {
	return self.write(src, file)
}

func (self *S3FileBackend) CopyTo(file File, to func(string) string) model.AppError {
	oldPath := file.GetPropertyString("location")
	newPath := to(oldPath)

	file.SetPropertyString("old_path", oldPath)

	params := &s3.CopyObjectInput{
		Bucket:     &self.bucket,
		CopySource: aws.String(fmt.Sprintf("%s/%s", self.bucket, oldPath)),
		Key:        aws.String(newPath),
	}

	_, err := self.svc.CopyObject(params)
	if err != nil {
		return model.NewInternalError("utils.file.s3.copy", err.Error())
	}

	file.SetPropertyString("location", newPath)

	return nil

}

func (self *S3FileBackend) write(src io.Reader, file File) (int64, model.AppError) {
	location, appErr := objectKey(self.GetStoreDirectory(file), file.GetStoreName())
	if appErr != nil {
		return 0, appErr
	}

	// A put replaces an object silently, and a store name is predictable from
	// what the client sends (a call record is "<call_id>_<call_id>_<name>"),
	// so never write over a key that is already taken.
	if _, err := self.svc.HeadObject(&s3.HeadObjectInput{Bucket: &self.bucket, Key: aws.String(location)}); err == nil {
		return 0, model.NewBadRequestError(ErrObjectExistsId, "name="+file.GetStoreName())
	}
	isEncrypted := file.IsEncrypted()

	params := &s3manager.UploadInput{
		Bucket: &self.bucket,
		Key:    aws.String(location),
	}

	if isEncrypted {
		params.Body = NewEncryptingReader(src, self.chipher)
	} else {
		params.Body = src
	}

	res, err := self.uploader.Upload(params)
	if err != nil {
		var apperr model.AppError
		if errors.As(err, &apperr) {
			return 0, apperr
		}

		var aerr awserr.Error
		if errors.As(err, &aerr) {
			originErr := aerr.OrigErr()
			if originErr != nil {
				switch originErr.(type) {
				case model.AppError:
					return 0, originErr.(model.AppError)
				default:
					return 0, model.NewInternalError("utils.file.s3.writing.origin", aerr.OrigErr().Error())
				}

			}

			return 0, model.NewInternalError("utils.file.s3.writing", aerr.Error())
		}

		return 0, model.NewInternalError("utils.file.s3.writing.unknown", err.Error())
	}

	file.SetPropertyString("location", location)
	wlog.Debug(fmt.Sprintf("[%s] create new file %s", self.name, res.Location))
	h, _ := self.svc.HeadObject(&s3.HeadObjectInput{
		Bucket: params.Bucket,
		Key:    params.Key,
	})

	s := file.GetSize()
	if h != nil && h.ContentLength != nil {
		s = *h.ContentLength
	}

	if isEncrypted {
		s, _ = EstimateOriginalSize(s)
	}

	return s, nil
}

func (self *S3FileBackend) Remove(file File) model.AppError {
	location := file.GetPropertyString("location")
	return self.remove(location)
}

func (self *S3FileBackend) remove(location string) model.AppError {
	_, err := self.svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &self.bucket,
		Key:    aws.String(location),
	})

	if err != nil {
		return model.NewInternalError("utils.file.s3.remove.app_error", err.Error())
	}

	return nil
}

func (self *S3FileBackend) RemoveFile(directory, name string) model.AppError {
	location, appErr := objectKey(directory, name)
	if appErr != nil {
		return appErr
	}

	_, err := self.svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &self.bucket,
		Key:    aws.String(location),
	})

	if err != nil {
		return model.NewInternalError("utils.file.s3.remove.app_error", err.Error())
	}
	return nil
}

func (self *S3FileBackend) Reader(file File, offset int64) (io.ReadCloser, model.AppError) {
	var rng *string = nil
	if offset > 0 {
		rng = aws.String("bytes=" + strconv.FormatInt(EstimateFirstBlockOffset(file, offset), 10) + "-")
	}

	params := &s3.GetObjectInput{
		Bucket: &self.bucket,
		Key:    aws.String(file.GetPropertyString("location")),
		Range:  rng,
	}

	out, err := self.svc.GetObject(params)
	if err != nil {
		return nil, model.NewInternalError("utils.file.s3.reader.app_error", err.Error())
	}

	if file.IsEncrypted() {
		return NewDecryptingReader(out.Body, self.chipher, offset), nil
	}

	return out.Body, nil
}
