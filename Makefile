.DEFAULT_GOAL	:= build

SHELL 	:= /bin/bash
PKG 		:= github.com/launchdarkly/terraform-provider-launchdarkly

# Pure Go sources (not vendored and not generated)
GOFILES		= $(shell find . -type f -name '*.go' -not -path "./vendor/*")
GODIRS		= $(shell go list -f '{{.Dir}}' ./... \
						| grep -vFf <(go list -f '{{.Dir}}' ./vendor/...))
# old start:
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
check: format.check vet lint

.PHONY: format
format: tools.goimports
	@echo "--> formatting code with 'goimports' tool"
	@goimports -local $(PKG) -w -l $(GOFILES)

.PHONY: format.check
format.check: tools.goimports
	@echo "--> checking code formatting with 'goimports' tool"
	@goimports -local $(PKG) -l $(GOFILES) | sed -e "s/^/\?\t/" | tee >(test -z)

.PHONY: vet
vet:
	@echo "--> checking code correctness with 'go vet' tool"
	@go vet ./...

.PHONY: lint
lint: tools.golint
	@echo "--> checking code style with 'golint' tool"
	@echo $(GODIRS) | xargs -n 1 golint | grep -v keys.go || true

#---------------
#-- tools
#---------------
.PHONY: tools tools.goimports tools.golint

tools: tools.goimports tools.golint

tools.goimports:
	@command -v goimports >/dev/null ; if [ $$? -ne 0 ]; then \
		echo "--> installing goimports"; \
		go get golang.org/x/tools/cmd/goimports; \
	fi

tools.golint:
	@command -v golint >/dev/null ; if [ $$? -ne 0 ]; then \
		echo "--> installing golint"; \
		go get -u golang.org/x/lint/golint; \
	fi