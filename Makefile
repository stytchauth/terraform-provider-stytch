default: fmt lint install generate

GO_BUILD_FLAGS = -trimpath

.PHONY: build
build:
	go build $(GO_BUILD_FLAGS) -v ./... 

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
	TEST_CMD := go test $(GO_BUILD_FLAGS)
endif

.PHONY: test
test:
	$(TEST_CMD) ./...

.PHONY: testacc
testacc:
	TF_ACC=1 $(TEST_CMD) ./internal/provider/resources/emailtemplate_test.go # ./...

.PHONY: toggle-local-management-go
toggle-local-management-go:
	./hack/toggle-dependency.sh stytch-management-go

# Initial setup:
# export STYTCH_WORKSPACE_KEY_ID=workspace-key-prod-1c7c5264-5e94-493b-9931-7a85e5e0f52d
# export STYTCH_WORKSPACE_KEY_SECRET=-QN1FMMVVqELfPcAKjrVWfiGqDuijQ5UU5lhpeF_LU29KbhY

# Failed to create email template w/ "invalid_workspace_domain", "Not found":
# export STYTCH_CUSTOM_DOMAIN=logangore.dev

