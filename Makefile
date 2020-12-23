MODULE   = $(shell env GO111MODULE=on $(GO) list -m)
DATE    ?= $(shell date +%FT%T%z)
VERSION ?= $(shell git describe --tags --always --dirty --match="v*" 2> /dev/null || \
			cat $(CURDIR)/.version 2> /dev/null || echo v0)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BRANCH  ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
PKGS     = $(or $(PKG),$(shell env GO111MODULE=on $(GO) list ./...))
TESTPKGS = $(shell env GO111MODULE=on $(GO) list -f \
			'{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' \
			$(PKGS))
GOLANGCI_LINT_CONFIG = $(CURDIR)/.golangci.yml
BIN      = $(CURDIR)/.bin
TARGETOS   ?= linux
TARGETARCH ?= amd64

GO      = go
TIMEOUT = 15
V = 0
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m▶\033[0m")

export GO111MODULE=on
export CGO_ENABLED=0
export GOPROXY=https://proxy.golang.org

.PHONY: all
all: fmt lintci test ; $(info $(M) building $(TARGETOS)/$(TARGETARCH) binary...) @ ## Build program binary
	$Q env GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) $(GO) build \
		-tags release \
		-ldflags '-X main.Version=$(VERSION) -X main.BuildDate=$(DATE) -X main.GitCommit=$(COMMIT) -X main.GitBranch=$(BRANCH)' \
		-o $(BIN)/$(basename $(MODULE)) main.go

# Tools

setup-tools: setup-golint setup-golangci-lint setup-gocov setup-gocov-xml setup-go2xunit setup-mockery

setup-golint:
	$(GO) get golang.org/x/lint/golint
setup-golangci-lint:
	$(GO) get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.33.0
setup-gocov:
	$(GO) get github.com/axw/gocov/...
setup-gocov-xml:
	$(GO) get github.com/AlekSi/gocov-xml
setup-go2xunit:
	$(GO) get github.com/tebeka/go2xunit
setup-mockery:
	$(GO) get github.com/vektra/mockery/v2/

GOLINT=golint
GOLANGCILINT=golangci-lint
GOCOV=gocov
GOCOVXML=gocov-xml
GO2XUNIT=go2xunit
GOMOCK=mockery

# Tests

TEST_TARGETS := test-default test-bench test-short test-verbose test-race
.PHONY: $(TEST_TARGETS) test-xml check test tests
test-bench:   ARGS=-run=__absolutelynothing__ -bench=. ## Run benchmarks
test-short:   ARGS=-short        ## Run only short tests
test-verbose: ARGS=-v            ## Run tests in verbose mode with coverage reporting
test-race:    ARGS=-race         ## Run tests with race detector
$(TEST_TARGETS): NAME=$(MAKECMDGOALS:test-%=%)
$(TEST_TARGETS): test
check test tests: fmt lintci ; $(info $(M) running $(NAME:%=% )tests...) @ ## Run tests
	$Q $(GO) test -timeout $(TIMEOUT)s $(ARGS) $(TESTPKGS)

test-xml: fmt lintci | setup-go2xunit ; $(info $(M) running xUnit tests...) @ ## Run tests with xUnit output
	$Q mkdir -p test
	$Q 2>&1 $(GO) test -timeout $(TIMEOUT)s -v $(TESTPKGS) | tee test/tests.output
	$(GO2XUNIT) -fail -input test/tests.output -output test/tests.xml

COVERAGE_MODE    = atomic
COVERAGE_PROFILE = $(COVERAGE_DIR)/profile.out
COVERAGE_XML     = $(COVERAGE_DIR)/coverage.xml
COVERAGE_HTML    = $(COVERAGE_DIR)/index.html
.PHONY: test-coverage test-coverage-tools
test-coverage-tools: | setup-gocov setup-gocov-xml
test-coverage: COVERAGE_DIR := $(CURDIR)/test/coverage.$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
test-coverage: fmt lintci test-coverage-tools ; $(info $(M) running coverage tests...) @ ## Run coverage tests
	$Q mkdir -p $(COVERAGE_DIR)
	$Q $(GO) test \
		-coverpkg=$$($(GO) list -f '{{ join .Deps "\n" }}' $(TESTPKGS) | \
					grep '^$(MODULE)/' | \
					tr '\n' ',' | sed 's/,$$//') \
		-covermode=$(COVERAGE_MODE) \
		-coverprofile="$(COVERAGE_PROFILE)" $(TESTPKGS)
	$Q $(GO) tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	$Q $(GOCOV) convert $(COVERAGE_PROFILE) | $(GOCOVXML) > $(COVERAGE_XML)

.PHONY: lint
lint: setup-golint; $(info $(M) running golint...) @ ## Run golint
	$Q $(GOLINT) -set_exit_status $(PKGS)

.PHONY: lintci
lintci: setup-golangci-lint; $(info $(M) running golangci-lint...) @ ## Run golangci-lint
	$Q $(GOLANGCILINT) run -v -c $(GOLANGCI_LINT_CONFIG) ./...

# generate test mock for interfaces
.PHONY: mockgen
mockgen: | setup-mockery ; $(info $(M) generating mocks...) @ ## Run mockery
	$Q $(GOMOCK) --dir internal/aws/autoscaling --inpackage --all
	$Q $(GOMOCK) --dir internal/aws/eventbridge --inpackage --all

.PHONY: fmt
fmt: ; $(info $(M) running gofmt...) @ ## Run gofmt on all source files
	$Q $(GO) fmt $(PKGS)

# Misc

.PHONY: clean
clean: ; $(info $(M) cleaning...)	@ ## Cleanup everything
	@rm -rf $(BIN)
	@rm -rf test/tests.* test/coverage.*

.PHONY: help
help:
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: version
version:
	@echo $(VERSION)
