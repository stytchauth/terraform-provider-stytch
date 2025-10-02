default: fmt lint install generate

GO_BUILD_FLAGS = -trimpath

.PHONY: build
build:
	go build $(GO_BUILD_FLAGS) -v ./... 

.PHONY: install
install: build
	go install -v ./...

# Prefer local semgrep to the dockerized version. The container image is the same one used by CI.
ifneq (,$(shell which semgrep))
SEMGREP := semgrep
else
SEMGREP := docker run -v "$$(pwd)":/src --workdir /src semgrep/semgrep:1.61.1 semgrep
endif

.PHONY: semgrep
semgrep:
	${SEMGREP} --config semgrep-rules --metrics=off
	$(MAKE) fmt lint

.PHONY: semgrep-ci
semgrep-ci:
	${SEMGREP} ci --config semgrep-rules --metrics=off

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
	TEST_CMD := go test $(GO_BUILD_FLAGS)
endif

.PHONY: test
test:
	$(TEST_CMD) ./...

.PHONY: testacc
testacc:
	TF_ACC=1 $(TEST_CMD) ./...

.PHONY: toggle-local-management-go
toggle-local-management-go:
	./hack/toggle-dependency.sh stytch-management-go/v3
