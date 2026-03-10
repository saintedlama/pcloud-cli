APPNAME = pcloud-cli
PACKAGE = github.com/saintedlama/${APPNAME}
MAIN=cmd/${APPNAME}/main.go

.PHONY: all clean fmt lint help
.DEFAULT_GOAL := help

all: build

clean:
	go clean
	rm ./${APPNAME} || true
	rm -rf ./release || true

build: ## Build pcloud-cli binary
	go build -o ${APPNAME} ${MAIN}

install: ## Install pcloud-cli binary
	go install ${PACKAGE}

fmt: ## Run gofmt linter
	@unformatted=$$(gofmt -l $$(find . -type f -name '*.go' -not -path './vendor/*')); \
	if [ -n "$$unformatted" ]; then \
		echo "$$unformatted"; \
		echo "^ improperly formatted go files"; \
		echo; \
		exit 1; \
	fi

lint: ## Run golint linter
	@for p in `go list ./... | grep -v /vendor/` ; do \
		if [ "`golint $$p | tee /dev/stderr`" ]; then \
			echo "^ golint errors!" && echo && exit 1; \
		fi \
	done

release: clean darwin linux ## Build pcloud-cli for Os X and Linux

darwin: ## Build pcloud-cli for Mac Os X
	GOOS=darwin GOARCH=386 go build -o ./release/${APPNAME}_darwin_386
	GOOS=darwin GOARCH=amd64 go build -o ./release/${APPNAME}_darwin_amd64

linux: ## Build pcloud-cli for Linux
	GOOS=linux GOARCH=386 go build -o ./release/${APPNAME}_linux_386
	GOOS=linux GOARCH=amd64 go build -o ./release/${APPNAME}_linux_amd64

windows: ## Build pcloud-cli for Windows
	GOOS=windows GOARCH=386 go build -o ./release/${APPNAME}_windows_386.exe
	GOOS=windows GOARCH=amd64 go build -o ./release/${APPNAME}_windows_amd64.exe

help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
