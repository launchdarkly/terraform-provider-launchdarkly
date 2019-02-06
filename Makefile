TEST ?= ./...

all: lint test testacc build

lint:
	go vet ./...
	# golint -set_exit_status ./...

test:
	go test $(TEST) -v $(TESTARGS)

testacc:
	TF_ACC=1 go test $(TEST) -v $(TESTARGS)

build:
	go build -o terraform-provider-launchdarkly

install: build
	install -d -m 755 ~/.terraform.d/plugins
	install terraform-provider-launchdarkly ~/.terraform.d/plugins

apply: build
	terraform init
	terraform $@

destroy: build
	terraform init
	terraform $@
