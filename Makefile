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

file_name = tokens-$(shell date +%s).json

## upload-coins/prod number=$1: generate coins and upload them to firestore
upload-coins/prod:
	go run scripts/go/generatecoins/generatecoins.go -o /tmp/ -f ${file_name} -n ${number}
	go run scripts/go/coinsfirestore/coinsfirestore.go -f /tmp/${file_name} -p ${PROJECT_ID} -c tokens

## upload-coins/dev number=$1: generate coins and upload them to firestore
upload-coins/dev:
	go run scripts/go/generatecoins/generatecoins.go -o /tmp/ -f ${file_name} -n ${number}
	go run scripts/go/coinsfirestore/coinsfirestore.go -f /tmp/${file_name} -p ${TEST_PROJECT_ID} -c tokens

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run/dev: run the api application in dev mode
run/dev:
	go run . -project-id=${TEST_PROJECT_ID}

## run/docker: run the docker container
run/docker:
	docker run -d --name token-tltv token-tltv:latest

## run/local: run locally with no token check
run/local:
	go run . -env=local -project-id=${TEST_PROJECT_ID}

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
	go test -race -vet=off ./... -test=unit -coverprofile=coverage.out

## audit/local: tidy dependencies and format, vet and test all code (race off)
audit/local:
	make audit
	make ci-lint
	make vuln
	go test -vet=off ./... -coverprofile=coverage.out
	go test ./... -test=integration -project-id=token-tltv-test
	go test ./... -test=end-to-end -project-id=token-tltv-test

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
	go build -ldflags=${linker_flags} -o=./bin/tltv ./
	GOOS=linux GOARCH=amd64 go build -ldflags=${linker_flags} -o=./bin/linux_amd64/tltv ./

## docker/local: build the token-tltv container for local use (does not connect to firestore)
docker/local:
	@echo 'Building container...'
	docker build -f docker/local/Dockerfile  --build-arg PROJECT_ID=${TEST_PROJECT_ID}  --tag token-tltv:latest .

## docker/dev: build the token-tltv container for local use (does not connect to firestore)
docker/dev:
	@echo 'Building container...'
	docker build -f docker/dev/Dockerfile  --tag token-tltv:latest .

## docker/cloud: build and push the token-tltv container to the cloud
docker/cloud:
	docker build -f docker/prod/Dockerfile --platform linux/amd64 --push -t ${REGION}-docker.pkg.dev/${PROJECT_ID}/token-tltv/token-tltv-443:latest .

## build/pack: build the talkliketv container using build pack
build/pack:
	@echo 'Building container with buildpack'
	pack build token-tltv --env "LINKER_FLAGS=${linker_flags}" --builder paketobuildpacks/builder-jammy-base

# ==================================================================================== #
# CLOUD
# ==================================================================================== #

## connect: connect to the cloud server
connect:
	ssh ${CLOUD_HOST_USERNAME}@${CLOUD_HOST_IP}