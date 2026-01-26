# Variables
BINARY_NAME=crawler
CMD_PATH=cmd/crawler/main.go

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run: run the application locally (loads .env automatically via godotenv)
.PHONY: run
run:
	go run $(CMD_PATH)

## build: build the binary to the /bin directory
.PHONY: build
build:
	go build -o bin/$(BINARY_NAME) $(CMD_PATH)

## test: run all tests
.PHONY: test
test:
	go test -v ./...

## lint: run golangci-lint (must be installed)
.PHONY: lint
lint:
	golangci-lint run

## clean: remove binary and temporary files
.PHONY: clean
clean:
	go clean
	rm -f bin/$(BINARY_NAME)

# ==================================================================================== #
# DOCKER
# ==================================================================================== #

## docker-up: start the crawler and database in Docker
.PHONY: docker-up
docker-up:
	docker-compose up --build

## docker-down: stop all containers
.PHONY: docker-down
docker-down:
	docker-compose down

## docker-logs: follow the logs of the running containers
.PHONY: docker-logs
docker-logs:
	docker-compose logs -f