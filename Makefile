include .envrc
## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

## run/app: run the application
.PHONY: run/app
run/app:
	CASA_DB_DSN=${CASA_DB_DSN} go run ./...

## db/psql: connect to the database using psql
.PHONY: db/psql
db/psql:
	psql ${CASA_DB_DSN}

## audit: tidy dependencies and format, vet and test all code
.PHONY: audit
audit:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

## build/app: build the application
.PHONY: build/app
build/app:
	@echo 'Building app...'
	go build -ldflags='-s' -o=./bin/casa ./*.go

## clean: clean build folder
.PHONY: clean
clean:
	rm -rf bin
