package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := fileservice.NewFileServiceClient(conn)

	f, err := os.Open("test.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// создаём стрим
	stream, err := client.UploadFile(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 1024*32) // 32KB chunks
	for {
		n, err := f.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		req := &fileservice.UploadFileRequest{
			FileName: "test.jpg",
			Chunk:    buf[:n],
		}
		if err := stream.Send(req); err != nil {
			log.Fatal(err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Файл загружен! ID:", resp.GetFileId())
}
