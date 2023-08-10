
TEST?=$$(go list ./...)
GOFMT_FILES?=$$(find . -name '*.go')
PKG_NAME=launchdarkly
REV:=$(shell git rev-parse HEAD | cut -c1-6)
LDFLAGS:=-ldflags="-X main.version=$(REV) -X github.com/launchdarkly/terraform-provider-launchdarkly/launchdarkly.version=$(REV)"

default: build

build: fmtcheck
	go install $(LDFLAGS)

test: fmtcheck
	go test -i $(TEST) || exit 1
	echo $(TEST) | \
		xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=4

testacc: fmtcheck
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 120m

testacc-with-retry: fmtcheck
	make testacc || make testacc

vet:
	@echo "go vet ."
	@go vet $$(go list ./...) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

fmt:
	gofmts -w $(GOFMT_FILES)
	gofmt -w $(GOFMT_FILES)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

errcheck:
	@sh -c "'$(CURDIR)/scripts/errcheck.sh'"

install-codegen:
	cd scripts/codegen && go install && cd ../..

generate: install-codegen
	go generate ./...

test-compile:
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./$(PKG_NAME)"; \
		exit 1; \
	fi
	go test -c $(TEST) $(TESTARGS)

.PHONY: build install apply test testacc testacc-with-retry vet fmt fmtcheck errcheck lint test-compile
