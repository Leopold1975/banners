BIN := "./bin"

generate:
	(which oapi-codegen > /dev/null) || \
	(go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest)	
	mkdir -p ./internal/banners/api/oapi 2>/dev/null || echo "ok, banners api dir already created"

	oapi-codegen -package oapi -generate types ./api/banners.v1.yaml > ./internal/banners/api/oapi/types.go
	oapi-codegen -package oapi -generate client ./api/banners.v1.yaml > ./internal/banners/api/oapi/client.go
	oapi-codegen -package oapi -generate chi-server ./api/banners.v1.yaml > ./internal/banners/api/oapi/server.go

	go mod tidy

build: generate
	@go build -o $(BIN)/banners ./cmd/banners/main.go 

run: build
	docker compose -f ./deployments/docker-compose.yaml up --build

down:
	docker compose -f ./deployments/docker-compose.yaml down

test:
	CGO_ENABLED=1 go test -v -race -timeout=30s ./...
