docs:
	cd ./publisher/ && swag init

lint:
	 golangci-lint run --fix ./benchmarking/... ./consumer/... ./publisher/... ./rabbit/...

tidy:
	go mod tidy
	cd ./benchmarking/ && go mod tidy
	cd ./consumer/ && go mod tidy
	cd ./publisher/ && go mod tidy
	cd ./rabbit/ && go mod tidy

all: docs lint tidy
