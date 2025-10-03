package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	stdlog "log"
	"os"

	fileservice "github.com/ladev74/protos/gen/go/file_service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"fileservice/internal/config"
	"fileservice/internal/logger"
)

type demo struct {
	client fileservice.FileServiceClient
	logger *zap.Logger
}

func main() {
	var method, id, imagePath, configPath string

	flag.StringVar(&configPath, "config_path", "", "Path to the config file")
	flag.StringVar(&method, "method", "", "Testing method")
	flag.StringVar(&id, "id", "", "File id")
	flag.StringVar(&imagePath, "image_path", "", "Path to the image")
	flag.Parse()

	if configPath == "" {
		stdlog.Fatal("config file must specify")
	}

	cfg, err := config.New(configPath)
	if err != nil {
		stdlog.Fatalf("failed to initialize config: %v", err)
	}

	log, err := logger.New(cfg.Env)
	if err != nil {
		stdlog.Fatalf("failed to initialize logger: %v", err)
	}

	if method == "" {
		log.Fatal("You must specify a method")
	}

	addr := fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("cannot connect to grpc server", zap.Error(err))
	}

	defer func() {
		err = conn.Close()
		if err != nil {
			log.Warn("failed to close grpc connection", zap.Error(err))
		}
	}()

	client := fileservice.NewFileServiceClient(conn)

	d := demo{
		client: client,
		logger: log,
	}

	switch method {
	case "upload":
		d.demoUpload(imagePath)
	case "get":
		d.demoGet(id)
	case "list":
		d.demoList()
	}
}

func (d *demo) demoUpload(imagePath string) {
	if imagePath == "" {
		d.logger.Fatal("demoUpload: image path is required")
	}

	f, err := os.Open(imagePath)
	if err != nil {
		d.logger.Fatal("demoUpload: cannot open file", zap.Error(err))
	}
	defer f.Close()

	stream, err := d.client.UploadFile(context.Background())
	if err != nil {
		d.logger.Fatal("demoUpload: cannot create upload stream", zap.Error(err))
	}

	buf := make([]byte, 32*1024)
	for {
		n, err := f.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			d.logger.Fatal("demoUpload: failed to read file", zap.Error(err))
		}

		req := &fileservice.UploadFileRequest{
			FileName: imagePath,
			Chunk:    buf[:n],
		}

		if err := stream.Send(req); err != nil {
			d.logger.Fatal("demoUpload: failed to send chunk", zap.Error(err))
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		d.logger.Fatal("demoUpload: failed to close and receive", zap.Error(err))
	}

	d.logger.Info("demoUpload: file uploaded successfully", zap.String("file_id", resp.GetFileId()))
}

func (d *demo) demoGet(fileID string) {
	if fileID == "" {
		d.logger.Fatal("demoGet: file ID is required")
	}

	stream, err := d.client.GetFile(context.Background(), &fileservice.GetFileRequest{FileId: fileID})
	if err != nil {
		d.logger.Fatal("demoGet: failed to start stream", zap.Error(err))
	}

	outFile, err := os.Create(fileID + ".jpg")
	if err != nil {
		d.logger.Fatal("demoGet: failed to create output file", zap.Error(err))
	}
	defer outFile.Close()

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			d.logger.Fatal("demoGet: failed to receive chunk", zap.Error(err))
		}

		if _, err := outFile.Write(resp.GetChunk()); err != nil {
			d.logger.Fatal("demoGet: failed to write chunk to file", zap.Error(err))
		}
	}

	d.logger.Info("demoGet: file downloaded successfully", zap.String("file_name", outFile.Name()))
}

func (d *demo) demoList() {
	resp, err := d.client.ListFiles(context.Background(), &fileservice.ListFilesRequest{
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		d.logger.Fatal("demoList: failed to list files", zap.Error(err))
	}

	d.logger.Info("demoList: files list retrieved", zap.Int("count", len(resp.Files)))
	for _, file := range resp.Files {
		fmt.Printf("id: %s, created_at: %s, updated_at: %s\n",
			file.Name, file.CreatedAt.AsTime(), file.UpdatedAt.AsTime())
	}
}
