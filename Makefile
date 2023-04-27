# If the first argument is "run"...
ifeq (run,$(firstword $(MAKECMDGOALS)))
  # use the rest as arguments for "run"
  RUN_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  # ...and turn them into do-nothing targets
  $(eval $(RUN_ARGS):;@:)
endif

.PHONY: docs_publisher
docs_publisher:
	cd ./publisher/ && swag init

.PHONY: lint
lint:
	 golangci-lint run --fix ./benchmarking/... ./consumer/... ./publisher/... ./rabbit/...

.PHONEY: tidy_benchmarking
tidy_benchmarking:
	cd ./benchmarking/ && go mod tidy

.PHONEY: tidy_consumer
tidy_consumer:
	cd ./consumer/ && go mod tidy

.PHONEY: tidy_publisher
tidy_publisher:
	cd ./publisher/ && go mod tidy

.PHONEY: tidy_rabbit
tidy_rabbit:
	cd ./rabbit/ && go mod tidy

.PHONY: tidy
tidy: tidy_benchmarking tidy_consumer tidy_publisher tidy_rabbit
	go mod tidy

.PHONY: run_publisher
run_publisher: docs_publisher tidy_publisher
	go run ./publisher/

.PHONY: all
all: docs lint tidy
