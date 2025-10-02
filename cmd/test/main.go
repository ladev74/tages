package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"google.golang.org/grpc"
)

func main() {
	var method, id string

	flag.StringVar(&method, "method", "", "Testing method")
	flag.StringVar(&id, "id", "", "File id")
	flag.Parse()
	//TODO: validate

	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := fileservice.NewFileServiceClient(conn)

	switch method {
	case "upload":
		testUpload(client)
	case "get":
		testGet(client, id)
	case "cList":
		concurrencyList(client)
	case "cUpload":
		cUpload(client)
	}
}

func uploadHold(client fileservice.FileServiceClient, id int, wg *sync.WaitGroup) {
	defer wg.Done()
	ctx := context.Background()
	stream, err := client.UploadFile(ctx)
	if err != nil {
		log.Printf("stream create err %d: %v\n", id, err)
		return
	}

	req := &fileservice.UploadFileRequest{
		FileName: fmt.Sprintf("file-%d.jpg", id),
		Chunk:    []byte("first-chunk"),
	}
	if err := stream.Send(req); err != nil {
		log.Printf("send err %d: %v\n", id, err)
		_ = stream.CloseSend()
		return
	}

	_ = stream.CloseSend()
	_, err = stream.CloseAndRecv()
	if err != nil && err != io.EOF {
		log.Printf("close recv err %d: %v\n", id, err)
		return
	}
	log.Printf("upload %d finished\n", id)
}

func cUpload(client fileservice.FileServiceClient) {
	var wg sync.WaitGroup
	concurrent := 5
	wg.Add(concurrent)
	for i := 0; i < concurrent; i++ {
		go uploadHold(client, i, &wg)
	}

	wg.Wait()
	fmt.Println("done")
}

func downloadFile(client fileservice.FileServiceClient, fileID string, idx int, wg *sync.WaitGroup) {
	defer wg.Done()

	stream, err := client.GetFile(context.Background(), &fileservice.GetFileRequest{
		FileId: fileID,
	})
	if err != nil {
		log.Printf("download %d failed to start: %v\n", idx, err)
		return
	}

	f, err := os.CreateTemp("", "downloaded_*.bin")
	if err != nil {
		log.Printf("download %d cannot create file: %v\n", idx, err)
		return
	}
	defer f.Close()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("download %d error receiving chunk: %v\n", idx, err)
			return
		}
		_, err = f.Write(chunk.GetChunk())
		if err != nil {
			log.Printf("download %d error writing chunk: %v\n", idx, err)
			return
		}
	}

	log.Printf("download %d finished successfully\n", idx)
}

func concurrencyList(client fileservice.FileServiceClient) {
	var wg sync.WaitGroup
	ctx := context.Background()

	concurrent := 10
	wg.Add(concurrent)

	for i := 0; i < concurrent; i++ {
		go func(ii int) {
			defer wg.Done()

			resp, err := client.ListFiles(ctx, &fileservice.ListFilesRequest{
				Limit:  10,
				Offset: 0,
			})
			if err != nil {
				log.Printf("list %d error: %v\n", ii, err)
				return
			}

			log.Printf("list %d got %d files\n", ii, len(resp.Files))
		}(i)
	}

	wg.Wait()
	log.Println("done")
}

func testUpload(client fileservice.FileServiceClient) {
	f, err := os.Open("test.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

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

		err = stream.Send(req)
		if err != nil {
			log.Fatal(err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Файл загружен! ID:", resp.GetFileId())
}

func testGet(client fileservice.FileServiceClient, id string) {
	stream, err := client.GetFile(context.Background(), &fileservice.GetFileRequest{
		FileId: id,
	})
	if err != nil {
		log.Fatal(err)
	}

	outFile, err := os.Create("downloaded_test.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		_, err = outFile.Write(resp.GetChunk())
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("Файл успешно скачан!")
}
