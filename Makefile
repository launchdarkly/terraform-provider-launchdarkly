BUILD := $(abspath ./build)
TOOLS_BIN := $(BUILD)/tools

.DEFAULT_GOAL	:= build

all:  check test build

.PHONY: build
build:
	@echo "--> building"
	@go build -o terraform-provider-launchdarkly

.PHONY: install
install: build
	install -d -m 755 ~/.terraform.d/plugins
	install terraform-provider-launchdarkly ~/.terraform.d/plugins

.PHONY: apply
apply: build
	terraform init
	terraform $@

.PHONY: clean
clean:
	@echo "--> cleaning compiled objects and binaries"
	@go clean  -i ./...

.PHONY: test
test:
	@echo "--> running unit tests"
	@go test ./launchdarkly -v $(TESTARGS)

.PHONY: testacc
testacc:
	@echo "--> running acceptance tests"
	TF_ACC=1 go test ./launchdarkly -v $(TESTARGS)

.PHONY: check
check: format.check lint

.PHONY: format
format: tools.golangci-lint
	@echo "--> formatting code with 'goimports' tool via golangci-lint"
	$(TOOLS_BIN)/golangci-lint run --disable-all --enable goimports --fix

.PHONY: format.check
format.check: tools.golangci-lint
	@echo "--> checking code formatting with 'goimports' tool via golangci-lint"
	$(TOOLS_BIN)/golangci-lint run --disable-all --enable goimports

.PHONY: lint
lint: tools.golangci-lint
	@echo "--> checking code style with golangci-lint"
	$(TOOLS_BIN)/golangci-lint run

#---------------
#-- tools
#---------------
.PHONY: tools.golangci-lint

tools.golangci-lint:
	GOBIN=$(TOOLS_BIN) go install -mod=vendor github.com/golangci/golangci-lint/cmd/golangci-lint