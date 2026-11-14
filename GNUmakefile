
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
		xargs -t -n4 go test $(TESTARGS) -timeout=90s -parallel=4

testacc: fmtcheck
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 120m -parallel=4

testacc-with-retry:
	make testacc || make testacc

# Cross-checks every TestAcc* function name against the test_case matrices in
# .github/workflows/test.yml and test-fork-pr.yml. Requires no credentials.
matrixcheck:
	go test ./$(PKG_NAME)/ -run TestCIMatrixCoversAcceptanceTests -count=1

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
	go generate ./launchdarkly/...
	go generate .

# Bump the LaunchDarkly API client to a published release. A major bump moves the
# module path (api-client-go/vN), so the script rewrites the import path in every
# file that references it, not just go.mod.
#  make update-api-client-go API_CLIENT_GO_VERSION=24.0.0
# Normally run for you by the release automation in ld-openapi-private, which
# opens the bump PR here after publishing a new client version.
update-api-client-go:
	./scripts/update_api_client_go.sh $(API_CLIENT_GO_VERSION)

test-compile:
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./$(PKG_NAME)"; \
		exit 1; \
	fi
	go test -c $(TEST) $(TESTARGS)

.PHONY: build install apply test testacc testacc-with-retry matrixcheck vet fmt fmtcheck errcheck lint test-compile
