default: help


build: ALWAYS  ## Build app
	go build -o arisa3
	@echo [make build] Done ✓


test:  ## Run unit tests
	# Ensure gotestsum is installed
	go install gotest.tools/gotestsum@v1.8.1

	gotestsum -f dots -- -failfast -covermode=count -coverprofile coverage.out ./...

	@go tool cover -func=coverage.out | grep 'total' | sed -e 's/\t\+/ /g'
	@echo [make test] Done ✓


install-dev: install tooling  ## Install for dev environments


install: ALWAYS  ## Install build dependencies
	go get
	@echo [make install] Done ✓


tooling: ALWAYS  ## Install development tooling
	@echo Installing golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.45.2
	golangci-lint --version
	@echo

	pip install pre-commit -q
	pre-commit install
	@echo

	go install gotest.tools/gotestsum@v1.8.1
	gotestsum --version
	@echo

	go install github.com/golang/mock/mockgen@v1.6.0
	mockgen -version

	@echo [make tooling] Done ✓


help:  ## Show this help
	@echo 'usage: make [target] ...'
	@echo ''
	@echo 'targets:'
	@egrep '^(.+)\:\ .*##\ (.+)' ${MAKEFILE_LIST} | sed 's/:.*##/#/' | column -t -c 2 -s '#'


.PHONY: ALWAYS
ALWAYS:
