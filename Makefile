.PHONY: build run deps tidy clean test coverage lint help default

GO_CMD=go
BIN_NAME=Carvana.VDI.DescriptionMapperAndValuations
COVERAGE_FILE=profile.cov
DOCKER_COMPOSE=docker compose
POSTGRES_DB=postgres-db
REDIS=redis
API=api

default: build

help:
	@echo 'Management commands for Carvana.VDI.DescriptionMapperAndValuations:'
	@echo
	@echo 'Usage:'
	@echo '    make build           Compile the project.'
	@echo '    make run             Run the project.'
	@echo '    make deps            Download dependencies.'
	@echo '    make tidy            Clean up go.mod.'
	@echo '    make clean           Clean generated files.'
	@echo '    make test            Run tests on a compiled project.'
	@echo '    make coverage        Run tests with a terminal coverage report.'
	@echo '    make coverage/html   Run tests with an HTML coverage report.'
	@echo '    make lint            Run linter.'
	@echo

build:
	${GO_CMD} build -o bin/${BIN_NAME}

run:
	${GO_CMD} run .

deps:
	${GO_CMD} mod download

tidy:
	${GO_CMD} mod tidy

clean:
	@test ! -e bin/${BIN_NAME} || rm bin/${BIN_NAME}
	@test ! -e ${COVERAGE_FILE} || rm ${COVERAGE_FILE}

test:
	${GO_CMD} test ./tests/...

coverage:
	${GO_CMD} test -cover -coverpkg=./... -covermode=count -coverprofile=${COVERAGE_FILE} ./...
	${GO_CMD} tool cover -func=${COVERAGE_FILE}

coverage/html:
	${GO_CMD} test -cover -coverpkg=./... -covermode=count -coverprofile=${COVERAGE_FILE} ./...
	${GO_CMD} tool cover -html=${COVERAGE_FILE}

lint:
	pre-commit run --all-files

generate:
	${GO_CMD} get github.com/99designs/gqlgen@v0.17.30
	go generate ./...

dc-logs:
	${DOCKER_COMPOSE} logs -f

dc-api-up:
	${DOCKER_COMPOSE} up -d ${API}

dc-postgres-up:
	${DOCKER_COMPOSE} up -d ${POSTGRES_DB}

dc-redis-up:
	${DOCKER_COMPOSE} up -d ${REDIS}

dc-down:
	${DOCKER_COMPOSE} down
