start-postgres:
	docker compose up -d postgres

start-minio:
	docker compose up -d minio

up-migration:
	go run cmd/migrator/main.go --config_path=config/local.yaml --migration_path=database/migrations

start-app:
	go run cmd/file_service/main.go --config_path=config/local.yaml

down:
	docker compose down

demo-upload:
	go run cmd/demo/main.go --config_path=config/local.yaml --method=upload --image_path=test.jpg

demo-get:
	go run cmd/demo/main.go --config_path=config/local.yaml --method=get --id=$(id)

demo-list:
	go run cmd/demo/main.go --config_path=config/local.yaml --method=list

all: start-postgres start-minio up-migration start-app