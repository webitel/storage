package interfaces

import "context"

type SynchronizerFilesInterface interface {
	Start(ctx context.Context)
	Stop()
}
