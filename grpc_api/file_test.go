package grpc_api

import (
	gogrpc "buf.build/gen/go/webitel/storage/grpc/go/_gogrpc"
	storage "buf.build/gen/go/webitel/storage/protocolbuffers/go"
	"context"
	"fmt"
	"github.com/webitel/storage/model"
	"google.golang.org/grpc"
	"os"
	"testing"
	"time"
)

var service = "10.10.10.201:8767"
var fileLoc = "/Users/ihor/work/storage/1/img.png"

func TestFile(t *testing.T) {
	uploadId := sendFile(nil)
	time.Sleep(time.Second)
	sendFile(uploadId)
	time.Sleep(time.Second)
	sendFile(uploadId)
	time.Sleep(time.Second)
	sendFile(uploadId)
	sendFile(uploadId)
	sendFile(uploadId)

}

func sendFile(uploadId *string) (newUploadId *string) {
	c, err := grpc.Dial(service, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))
	check(err)

	f, err := os.Open(fileLoc)
	check(err)
	defer f.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()

	api := gogrpc.NewFileServiceClient(c)
	s, err := api.SafeUploadFile(ctx)
	check(err)

	if uploadId != nil {
		err = s.Send(&storage.SafeUploadFileRequest{
			Data: &storage.SafeUploadFileRequest_UploadId{
				UploadId: *uploadId,
			},
		})
	} else {
		err = s.Send(&storage.SafeUploadFileRequest{
			Data: &storage.SafeUploadFileRequest_Metadata_{
				Metadata: &storage.SafeUploadFileRequest_Metadata{
					DomainId:       1,
					Name:           "3131231.png",
					MimeType:       "image/png",
					Uuid:           "blabla",
					StreamResponse: false,
					ProfileId:      221,
				},
			},
		})
	}

	check(err)
	rcv, err := s.Recv()
	check(err)
	switch r := rcv.Data.(type) {
	case *storage.SafeUploadFileResponse_Part_:
		f.Seek(r.Part.Size, 0)
		newUploadId = model.NewString(r.Part.UploadId)
		fmt.Println(r.Part)
	}

	buf := make([]byte, 4*1027)
	i := 0
	var n int
	for {
		n, err = f.Read(buf)
		i++

		if i == 50 {
			cancel()
			return
		}

		if n == 0 {
			break
		}
		err = s.Send(&storage.SafeUploadFileRequest{
			Data: &storage.SafeUploadFileRequest_Chunk{
				Chunk: buf[:n],
			},
		})
		check(err)
	}
	err = s.CloseSend()
	check(err)

	rcv, err = s.Recv()
	check(err)

	fmt.Println(rcv)
	return
}

func check(err error) {
	if err != nil {
		panic(err.Error())
	}
}
