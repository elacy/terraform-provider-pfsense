HOSTNAME    = registry.terraform.io
NAMESPACE   = elacy
NAME        = pfsense
BINARY      = terraform-provider-${NAME}
VERSION     = 0.1.0
OS_ARCH     = $(shell go env GOOS)_$(shell go env GOARCH)

default: build

build:
	go build -o ${BINARY}

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

test:
	go test ./...

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

fmt:
	gofmt -w .

vet:
	go vet ./...

generate:
	go generate ./...

.PHONY: build install test testacc fmt vet generate
