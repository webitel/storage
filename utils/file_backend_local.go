package utils

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/webitel/wlog"

	"github.com/webitel/storage/model"
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
	ErrPathEscapeId      = "utils.file.locally.path_escape"
)

// localPath joins parts under the backend's directory and refuses a path
// that leaves it. Store names are built from what the client sends (an
// upload's ?name=, a gRPC request's name or uuid), so "9_<id>_/../../x" must
// not reach the file system above the root.
func (self *LocalFileBackend) localPath(parts ...string) (string, model.AppError) {
	root := filepath.Clean(self.directory)
	full := filepath.Join(append([]string{root}, parts...)...)
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", model.NewBadRequestError(ErrPathEscapeId, fmt.Sprintf("path %q leaves the storage directory", filepath.Join(parts...)))
	}
	return full, nil
}

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
	return self.write(src, file, directory, file.IsEncrypted())
}

func (self *LocalFileBackend) write(src io.Reader, file File, directory string, encrypt bool) (int64, model.AppError) {
	root, appErr := self.localPath(directory)
	if appErr != nil {
		return 0, appErr
	}
	allPath, appErr := self.localPath(directory, file.GetStoreName())
	if appErr != nil {
		return 0, appErr
	}

	fi, _ := os.Stat(allPath)
	if fi != nil && fi.Size() > 0 {
		file.SetPropertyString("directory", directory)
		return 0, model.NewBadRequestError(ErrFileWriteExistsId, "name="+file.GetStoreName())
	}

	if err := os.MkdirAll(root, 0o774); err != nil {
		return 0, model.NewInternalError("utils.file.locally.create_dir.app_error", err.Error())
	}

	fw, err := os.OpenFile(allPath, os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return 0, model.NewInternalError("utils.file.locally.writing.app_error", err.Error())
	}

	defer fw.Close()

	var written int64

	if encrypt {
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

	if encrypt {
		written, _ = EstimateOriginalSize(written)
	}
	file.SetPropertyString("directory", directory)
	wlog.Debug(fmt.Sprintf("create new file %s", allPath))

	return written, nil
}

func (self *LocalFileBackend) Remove(file File) model.AppError {
	fullPath, appErr := self.localPath(file.GetPropertyString("directory"), file.GetStoreName())
	if appErr != nil {
		return appErr
	}
	if err := os.Remove(fullPath); err != nil {
		e, ok := err.(*os.PathError)
		if ok && e.Err == syscall.ENOENT {
			return model.NewNotFoundError("utils.file.locally.removing.not_found", err.Error())
		} else {
			return model.NewInternalError("utils.file.locally.removing.app_error", err.Error())
		}
	}
	return nil
}

func (self *LocalFileBackend) CopyTo(file File, to func(string) string) model.AppError {
	s, err := self.Reader(file, 0)
	if err != nil {
		return err
	}
	defer s.Close()

	oldPath := file.GetPropertyString("directory")
	newPath := to(oldPath)

	file.SetPropertyString("old_path", oldPath)

	_, err = self.write(s, file, newPath, false)
	if err != nil {
		return err
	}
	file.SetPropertyString("directory", newPath)

	return nil
}

func (self *LocalFileBackend) RemoveFile(directory, name string) model.AppError {
	fullPath, appErr := self.localPath(directory, name)
	if appErr != nil {
		return appErr
	}

	if err := os.Remove(fullPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return model.NewNotFoundError(
				"utils.file.locally.removing.not_found",
				fmt.Sprintf("file %q not found", fullPath),
			)
		}
		return model.NewInternalError(
			"utils.file.locally.removing.app_error",
			fmt.Sprintf("failed to remove file %q: %v", fullPath, err),
		)
	}
	return nil
}

func (self *LocalFileBackend) Reader(file File, offset int64) (io.ReadCloser, model.AppError) {
	fullPath, appErr := self.localPath(file.GetPropertyString("directory"), file.GetStoreName())
	if appErr != nil {
		return nil, appErr
	}

	f, err := os.Open(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, model.NewNotFoundError(
				"api.file.reader.reading_local.not_found",
				fmt.Sprintf("file %q not found", fullPath),
			)
		}
		return nil, model.NewInternalError(
			"api.file.reader.reading_local.app_error",
			fmt.Sprintf("failed to open file %q: %v", fullPath, err),
		)
	}

	if offset > 0 {
		if _, err := f.Seek(EstimateFirstBlockOffset(file, offset), io.SeekStart); err != nil {
			f.Close()
			return nil, model.NewInternalError(
				"api.file.reader.reading_local.seek_error",
				fmt.Sprintf("failed to seek to offset %d in file %q: %v", offset, fullPath, err),
			)
		}
	}

	if file.IsEncrypted() {
		return NewDecryptingReader(f, self.chipher, offset), nil
	}
	return f, nil
}
