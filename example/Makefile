GO                 ?= go
GO_BUILD_SRC        = $(shell find . -name \*.go)
GO_BUILD_TARGET     = example
GO_GENERATE_SRC     = $(shell git grep -l '//go:generate')
GO_GENERATE_TARGET  = config/config_deepcopy.go

# -- default -------------------------------------------------------------------

.PHONY: all test

all: go-build

test: go-test

# -- go ------------------------------------------------------------------------

.PHONY: go-build go-run go-generate go-test

go-build: go-generate $(GO_BUILD_TARGET)

$(GO_BUILD_TARGET): $(GO_BUILD_SRC)
	$(GO) build -o $(GO_BUILD_TARGET) .

go-run: go-generate
	$(GO) run .

go-test: go-generate
	$(GO) test -v ./...

go-generate: $(GO_GENERATE_TARGET)

$(GO_GENERATE_TARGET): $(GO_GENERATE_SRC)
	$(GO) generate ./...
