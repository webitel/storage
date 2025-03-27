package utils

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/wlog"
)

const (
	SelCDN        = "selcdn.ru"
	GoogleStorage = "storage.googleapis.com"
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

func (self *S3FileBackend) TestConnection() engine.AppError {
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
		return engine.NewInternalError("utils.file.s3.test_connection.app_error", err.Error())
	}

	self.sess = sess
	self.svc = s3.New(sess)
	self.uploader = s3manager.NewUploader(sess)

	return nil
}

func (self *S3FileBackend) Write(src io.Reader, file File) (int64, engine.AppError) {
	return self.write(src, file)
}

func (self *S3FileBackend) write(src io.Reader, file File) (int64, engine.AppError) {
	directory := self.GetStoreDirectory(file)
	location := path.Join(directory, file.GetStoreName())

	params := &s3manager.UploadInput{
		Bucket: &self.bucket,
		Key:    aws.String(location),
		Body:   src,
	}

	res, err := self.uploader.Upload(params)

	if err != nil {
		if _, ok := err.(awserr.Error); ok {
			err = err.(awserr.Error).OrigErr()
		}
		switch e := err.(type) {
		case engine.AppError:
			return 0, e
		default:
			if err != nil {
				return 0, engine.NewInternalError("utils.file.s3.writing.app_error", err.Error())
			}
		}
	}
	file.SetPropertyString("location", location)
	wlog.Debug(fmt.Sprintf("[%s] create new file %s", self.name, res.Location))

	h, _ := self.svc.HeadObject(&s3.HeadObjectInput{
		Bucket: params.Bucket,
		Key:    params.Key,
	})

	// TODO fixme
	s := file.GetSize()
	if h != nil && h.ContentLength != nil {
		s = *h.ContentLength
	}

	return s, nil
}

func (self *S3FileBackend) Remove(file File) engine.AppError {
	directory := self.GetStoreDirectory(file)
	location := path.Join(directory, file.GetStoreName())

	_, err := self.svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &self.bucket,
		Key:    aws.String(location),
	})

	if err != nil {
		return engine.NewInternalError("utils.file.s3.remove.app_error", err.Error())
	}

	return nil
}

func (self *S3FileBackend) RemoveFile(directory, name string) engine.AppError {
	location := path.Join(directory, name)

	_, err := self.svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &self.bucket,
		Key:    aws.String(location),
	})

	if err != nil {
		return engine.NewInternalError("utils.file.s3.remove.app_error", err.Error())
	}
	return nil
}

func (self *S3FileBackend) Reader(file File, offset int64) (io.ReadCloser, engine.AppError) {
	var rng *string = nil
	if offset > 0 {
		rng = aws.String("bytes=" + strconv.FormatInt(offset, 10) + "-")
	}

	params := &s3.GetObjectInput{
		Bucket: &self.bucket,
		Key:    aws.String(file.GetPropertyString("location")),
		Range:  rng,
	}

	out, err := self.svc.GetObject(params)
	if err != nil {
		return nil, engine.NewInternalError("utils.file.s3.reader.app_error", err.Error())
	}

	return out.Body, nil
}
