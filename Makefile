default: fmt lint install generate

.PHONY: build
build:
	go build -v ./... 

.PHONY: install
install: build
	go install -v ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: generate
generate:
	cd tools; go generate ./...

.PHONY: fmt
fmt:
	gofmt -s -w -e .

TEST_CMD := gotestsum 
ifeq (, $(shell command -v gotestsum))
	TEST_CMD := go test
endif

.PHONY: test
test:
	$(TEST_CMD) ./...

.PHONY: testacc
testacc:
	TF_ACC=1 $(TEST_CMD) ./...
