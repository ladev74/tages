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

all: start-postgres start-minio up-migration start-app