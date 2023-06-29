package utils

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"syscall"

	engine "github.com/webitel/engine/model"
	"github.com/webitel/wlog"
)

type LocalFileBackend struct {
	BaseFileBackend
	pathPattern string
	directory   string
	name        string
}

const (
	ErrFileWriteExistsId = "utils.file.locally.exists.app_error"
	ErrMaxLimitId        = "utils.file.locally.writing.limit"
)

func (self *LocalFileBackend) Name() string {
	return self.name
}

func (self *LocalFileBackend) GetStoreDirectory(domain int64) string {
	return path.Join(parseStorePattern(self.pathPattern, domain))
}

func (self *LocalFileBackend) TestConnection() engine.AppError {
	return nil
}

func (self *LocalFileBackend) Write(src io.Reader, file File) (int64, engine.AppError) {
	directory := self.GetStoreDirectory(file.Domain())
	root := path.Join(self.directory, directory)
	allPath := path.Join(root, file.GetStoreName())
	var err error

	_, err = os.Stat(allPath)
	if !os.IsNotExist(err) {
		file.SetPropertyString("directory", directory)
		return 0, engine.NewBadRequestError(ErrFileWriteExistsId, "root="+root+" name="+file.GetStoreName())
	}

	if err = os.MkdirAll(root, 0774); err != nil {
		return 0, engine.NewInternalError("utils.file.locally.create_dir.app_error", "root="+root+", err="+err.Error())
	}

	fw, err := os.OpenFile(allPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return 0, engine.NewInternalError("utils.file.locally.writing.app_error", err.Error())
	}

	defer fw.Close()
	written, err := io.Copy(fw, src)
	if err != nil {
		if err == ErrorMaxLimit {
			os.Remove(allPath)
			return 0, engine.NewBadRequestError(ErrMaxLimitId, err.Error())
		}
		return written, engine.NewInternalError("utils.file.locally.writing.app_error", err.Error())
	}

	self.setWriteSize(written)
	file.SetPropertyString("directory", directory)
	wlog.Debug(fmt.Sprintf("create new file %s", allPath))

	return written, nil
}

func (self *LocalFileBackend) Remove(file File) engine.AppError {
	if err := os.Remove(path.Join(self.directory, file.GetPropertyString("directory"), file.GetStoreName())); err != nil {
		e, ok := err.(*os.PathError)
		if ok && e.Err == syscall.ENOENT {
			return engine.NewNotFoundError("utils.file.locally.removing.not_found", err.Error())
		} else {
			return engine.NewInternalError("utils.file.locally.removing.app_error", err.Error())
		}
	}
	return nil
}

func (self *LocalFileBackend) RemoveFile(directory, name string) engine.AppError {
	if err := os.Remove(path.Join(self.directory, directory, name)); err != nil {
		return engine.NewInternalError("utils.file.locally.removing.app_error", err.Error())
	}
	return nil
}

func (self *LocalFileBackend) Reader(file File, offset int64) (io.ReadCloser, engine.AppError) {
	if f, err := os.Open(filepath.Join(self.directory, file.GetPropertyString("directory"), file.GetStoreName())); err != nil {
		return nil, engine.NewInternalError("api.file.reader.reading_local.app_error", err.Error())
	} else {
		if offset > 0 {
			f.Seek(offset, 0)
		}
		return f, nil
	}
}
