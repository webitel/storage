package interfaces

import "context"

type UploadRecordingsFilesInterface interface {
	Start(ctx context.Context)
	Stop()
}
