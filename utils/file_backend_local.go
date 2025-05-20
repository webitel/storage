package utils

import (
	"fmt"
	"github.com/webitel/storage/model"
	"io"
	"os"
	"path"
	"path/filepath"
	"syscall"

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

func (self *LocalFileBackend) GetStoreDirectory(f File) string {
	return path.Join(parseStorePattern(self.pathPattern, f))
}

func (self *LocalFileBackend) TestConnection() model.AppError {
	return nil
}

func (self *LocalFileBackend) Write(src io.Reader, file File) (int64, model.AppError) {
	directory := self.GetStoreDirectory(file)
	root := path.Join(self.directory, directory)
	allPath := path.Join(root, file.GetStoreName())
	isEncrypted := file.IsEncrypted()

	fi, _ := os.Stat(allPath)
	if fi != nil && fi.Size() > 0 {
		file.SetPropertyString("directory", directory)
		return 0, model.NewBadRequestError(ErrFileWriteExistsId, "name="+file.GetStoreName())
	}

	if err := os.MkdirAll(root, 0774); err != nil {
		return 0, model.NewInternalError("utils.file.locally.create_dir.app_error", err.Error())
	}

	fw, err := os.OpenFile(allPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return 0, model.NewInternalError("utils.file.locally.writing.app_error", err.Error())
	}

	defer fw.Close()

	var written int64

	if isEncrypted {
		written, err = io.Copy(fw, NewEncryptingReader(src, self.chipher))
	} else {
		written, err = io.Copy(fw, src)
	}

	if err != nil {
		os.Remove(allPath)
		switch err.(type) {
		case model.AppError:
			return 0, err.(model.AppError)
		default:
			return written, model.NewInternalError("utils.file.locally.writing.app_error", err.Error())
		}
	}

	self.setWriteSize(written)

	if isEncrypted {
		written, _ = EstimateOriginalSize(written)
	}
	file.SetPropertyString("directory", directory)
	wlog.Debug(fmt.Sprintf("create new file %s", allPath))

	return written, nil
}

func (self *LocalFileBackend) Remove(file File) model.AppError {
	if err := os.Remove(path.Join(self.directory, file.GetPropertyString("directory"), file.GetStoreName())); err != nil {
		e, ok := err.(*os.PathError)
		if ok && e.Err == syscall.ENOENT {
			return model.NewNotFoundError("utils.file.locally.removing.not_found", err.Error())
		} else {
			return model.NewInternalError("utils.file.locally.removing.app_error", err.Error())
		}
	}
	return nil
}

func (self *LocalFileBackend) RemoveFile(directory, name string) model.AppError {
	if err := os.Remove(path.Join(self.directory, directory, name)); err != nil {
		return model.NewInternalError("utils.file.locally.removing.app_error", "Encountered an error opening a reader from local server file storage")
	}
	return nil
}

func (self *LocalFileBackend) Reader(file File, offset int64) (io.ReadCloser, model.AppError) {
	if f, err := os.Open(filepath.Join(self.directory, file.GetPropertyString("directory"), file.GetStoreName())); err != nil {
		return nil, model.NewInternalError("api.file.reader.reading_local.app_error", "Encountered an error opening a reader from local server file storage")
	} else {

		if offset > 0 {
			f.Seek(EstimateFirstBlockOffset(file, offset), io.SeekStart)
		}

		if file.IsEncrypted() {
			return NewDecryptingReader(f, self.chipher, offset), nil
		}
		return f, nil
	}
}
