APPNAME = pcloud-cli
PACKAGE = github.com/saintedlama/${APPNAME}

.PHONY: all clean build install fmt lint help
.DEFAULT_GOAL := help

all: build

clean:
	go clean
	rm ./${APPNAME} || true

build: ## Build pcloud-cli binary
	go build -o ${APPNAME} .

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

help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
