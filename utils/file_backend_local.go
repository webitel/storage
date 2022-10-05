package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"syscall"

	"github.com/webitel/storage/model"
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
)

func (self *LocalFileBackend) Name() string {
	return self.name
}

func (self *LocalFileBackend) GetStoreDirectory(domain int64) string {
	return path.Join(parseStorePattern(self.pathPattern, domain))
}

func (self *LocalFileBackend) TestConnection() *model.AppError {
	return nil
}

func (self *LocalFileBackend) Write(src io.Reader, file File) (int64, *model.AppError) {
	directory := self.GetStoreDirectory(file.Domain())
	root := path.Join(self.directory, directory)
	allPath := path.Join(root, file.GetStoreName())
	var err error

	_, err = os.Stat(allPath)
	if !os.IsNotExist(err) {
		file.SetPropertyString("directory", directory)
		return 0, model.NewAppError("WriteFile", ErrFileWriteExistsId, nil, "root="+root+" name="+file.GetStoreName(), http.StatusBadRequest)
	}

	if err = os.MkdirAll(root, 0774); err != nil {
		return 0, model.NewAppError("WriteFile", "utils.file.locally.create_dir.app_error", nil, "root="+root+", err="+err.Error(), http.StatusInternalServerError)
	}

	fw, err := os.OpenFile(allPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return 0, model.NewAppError("WriteFile", "utils.file.locally.writing.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	defer fw.Close()
	written, err := io.Copy(fw, src)
	if err != nil {
		return written, model.NewAppError("WriteFile", "utils.file.locally.writing.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	self.setWriteSize(written)
	file.SetPropertyString("directory", directory)
	wlog.Debug(fmt.Sprintf("create new file %s", allPath))

	return written, nil
}

func (self *LocalFileBackend) Remove(file File) *model.AppError {
	if err := os.Remove(path.Join(self.directory, file.GetPropertyString("directory"), file.GetStoreName())); err != nil {
		e, ok := err.(*os.PathError)
		if ok && e.Err == syscall.ENOENT {
			return model.NewAppError("RemoveFile", "utils.file.locally.removing.not_found", nil, err.Error(), http.StatusNotFound)
		} else {
			return model.NewAppError("RemoveFile", "utils.file.locally.removing.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}
	return nil
}

func (self *LocalFileBackend) RemoveFile(directory, name string) *model.AppError {
	if err := os.Remove(path.Join(self.directory, directory, name)); err != nil {
		return model.NewAppError("RemoveFile", "utils.file.locally.removing.app_error", nil, err.Error(), http.StatusInternalServerError)
	}
	return nil
}

func (self *LocalFileBackend) Reader(file File, offset int64) (io.ReadCloser, *model.AppError) {
	if f, err := os.Open(filepath.Join(self.directory, file.GetPropertyString("directory"), file.GetStoreName())); err != nil {
		return nil, model.NewAppError("Reader", "api.file.reader.reading_local.app_error", nil, err.Error(), http.StatusInternalServerError)
	} else {
		if offset > 0 {
			f.Seek(offset, 0)
		}
		return f, nil
	}
}
