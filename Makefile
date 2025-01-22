# Include variables from the .envrc file
ifneq (,$(wildcard ./.envrc))
    include .envrc
endif

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

## copy-hooks: adds script to run before git push
copy-hooks:
	chmod +x scripts/hooks/*
	cp -r scripts/hooks .git/.

## expvar: add environment variable required for testing
expvar:
	eval $(cat .envrc)

## generate: generate code from specs
generate:
	go generate ./...

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run: run the api application
run:
	go run .

## run: run the docker container
run/docker:
	docker run -d --name token-tltv token-tltv:latest

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

audit:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	@echo 'Running tests...'

## audit/pipeline: tidy dependencies and format, vet and test all code (race on)
audit/pipeline:
	make audit
	go test -race -vet=off ./... -coverprofile=coverage.out

## audit/local: tidy dependencies and format, vet and test all code (race off)
audit/local:
	make audit
	make report
	make ci-lint
	make vuln

## staticcheck:  detect bugs, suggest code simplifications, and point out dead code
staticcheck:
	staticcheck ./...

## coverage
coverage:
	go tool cover -func coverage.out \
	| grep "total:" | awk '{print ((int($$3) > 80) != 1) }'

## coverage report
report:
	go test -vet=off ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o cover.html

install-golang-ci:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

ci-lint: install-golang-ci
	golangci-lint run

install-govulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@latest

vuln: install-govulncheck
	govulncheck ./*.go

# ==================================================================================== #
# BUILD
# ==================================================================================== #

current_time = $(shell date +"%Y-%m-%dT%H:%M:%S%Z")
git_description = $(shell git describe --always --dirty --tags --long)
linker_flags = '-s -X main.buildTime=${current_time} -X main.version=${git_description}'

## build: build the cmd/api application
build:
	@echo 'Building api...'
	go build -ldflags=${linker_flags} -o=./bin/tltv ./api
	GOOS=linux GOARCH=amd64 go build -ldflags=${linker_flags} -o=./bin/linux_amd64/tltv ./api

## build/docker: build the talkliketv container
build/docker:
	@echo 'Building container...'
	docker build --build-arg LINKER_FLAGS=${linker_flags} --tag token-tltv:latest .

## build/pack: build the talkliketv container using build pack
build/pack:
	@echo 'Building container with buildpack'
	pack build token-tltv --env "LINKER_FLAGS=${linker_flags}" --builder paketobuildpacks/builder-jammy-base