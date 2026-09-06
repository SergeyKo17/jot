.PHONY: build test lint create-migration

build:
	go build ./cmd/jot/

fmt:
	gofumpt -w .

create-migration:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: specify the migration name"; \
		echo "Ex: make create-migration NAME=create_memories"; \
		exit 1; \
	fi
	@mkdir -p migrations
	@TIMESTAMP=$$(date +%Y%m%d%H%M%S); \
	FILENAME="migrations/$${TIMESTAMP}_$(NAME).sql"; \
	echo "-- +goose Up" > $$FILENAME; \
	echo "" >> $$FILENAME; \
	echo "-- +goose Down" >> $$FILENAME; \
	echo "Migration created: $$FILENAME"

lint:
	golangci-lint run ./...

test:
	go test ./... -count=1