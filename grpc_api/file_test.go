package grpc_api

import (
	"context"
	"fmt"
	"github.com/webitel/storage/gen/storage"
	"github.com/webitel/storage/model"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"testing"
	"time"
)

var service = "10.10.10.25:8767"
var testFolder = "../test_data"

func TestFile(t *testing.T) {
	downloadFile()
	var uploadId *string
	fileLoc := testFolder + "/2.mp4"
	uploadId = sendFile(uploadId, fileLoc)
	for {
		time.Sleep(time.Millisecond * 500)
		uploadId = sendFile(uploadId, fileLoc)
		if uploadId == nil {
			fmt.Println("OK")
			return
		}
		fmt.Println("send")
	}
	return
	for uploadId = sendFile(uploadId, fileLoc); uploadId != nil; {
		fmt.Println("send")
		time.Sleep(time.Millisecond * 100)
	}

}

func downloadFile() {
	c, err := grpc.Dial(service, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))
	check(err)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()

	api := storage.NewFileServiceClient(c)

	fetchThumbnail := true

	s, err := api.DownloadFile(ctx, &storage.DownloadFileRequest{
		Id:             107167,
		DomainId:       1,
		Metadata:       true,
		Offset:         0,
		BufferSize:     0,
		FetchThumbnail: fetchThumbnail,
	})
	check(err)

	meta, err := s.Recv()
	check(err)
	if meta == nil {
		log.Fatalln("metadata is empty")
	}
	metadata := meta.Data.(*storage.StreamFile_Metadata_).Metadata

	fileLoc := testFolder + "/" + model.NewId()[:5]
	if fetchThumbnail && metadata.Thumbnail != nil {
		fileLoc += ".png"
	} else {
		fileLoc += ".mp4"
	}

	file, err := os.OpenFile(fileLoc, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	defer file.Close()

	for {
		msg, err := s.Recv()
		if err != nil {
			break
		}
		switch msg.Data.(type) {
		case *storage.StreamFile_Chunk:
			file.Write(msg.GetChunk())
		default:
			panic(1)
		}
	}
}

func sendFile(uploadId *string, fileLoc string) (newUploadId *string) {
	d := grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		var d net.Dialer
		c, err := d.Dial("tcp", s)
		if err != nil {
			return nil, err
		}
		tcpConn := c.(*net.TCPConn)
		//err = tcpConn.SetWriteBuffer(1 * 1024 * 1024)
		//if err != nil {
		//	return nil, err
		//}
		//
		//err = tcpConn.SetReadBuffer(1 * 1024 * 1024)
		//if err != nil {
		//	return nil, err
		//}
		//err = tcpConn.SetNoDelay(true)
		//if err != nil {
		//	return nil, err
		//}

		return tcpConn, nil
	})
	c, err := grpc.Dial(service, d, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))
	check(err)

	stats, err := os.Stat(fileLoc)
	check(err)

	f, err := os.Open(fileLoc)
	check(err)
	defer f.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()

	api := storage.NewFileServiceClient(c)
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
					DomainId: 1,
					Name:     stats.Name(),
					//MimeType: "image/png",
					MimeType:          "audio/mp4",
					Uuid:              "blabla",
					StreamResponse:    false,
					ProfileId:         221,
					GenerateThumbnail: true,
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
		if err != nil {
			break
		}
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
	newUploadId = nil
	return
}

func check(err error) {
	if err != nil {
		panic(err.Error())
	}
}
