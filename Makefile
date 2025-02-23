VERSION := `git describe --abbrev=0 --tags || echo "0.0.0"`
BUILD := `git rev-parse --short HEAD`

start:
	DISABLE_CORS=true go run cmd/server/main.go

build-snapshot:
	@goreleaser build --snapshot --clean

.PHONY: test
test:
ifeq ($(SKIP_INTEGRATION), true)
	@echo "Running unit tests only..."
	go test ./... -short -v -race -count 1
else
	@echo "Running all tests, including integration tests..."
	go test ./... -v -race -count 1
endif

test-coverage:	
	go test -v -race -count 1 -timeout 30s -tags='!integration' -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html

test-integration:
	go test -v -race -count 1 -timeout 30s -tags='integration' ./...

lint:
	golangci-lint run -v

format:
	gofumpt -l -w .

tidy-modules:
	go mod tidy && go mod vendor

count-lines:
	gocloc --not-match-d vendor .

.PHONY: bootstrap
bootstrap: ## Install tooling
	@go install $$(go list -e -f '{{join .Imports " "}}' ./internal/tools/tools.go)

.PHONY: check
check: vet staticcheck unparam semgrep check-fmt check-codegen check-gomod

.PHONY: staticcheck
staticcheck:
	@# Ignore below checks because of oapi-codegen generated files
	@staticcheck -checks=inherit,-ST1005,-SA1029,-SA4006,-SA1019 ./...

.PHONY: vet
vet:
	@go vet -mod=vendor ./...

.PHONY: unparam
unparam:
	@unparam ./...

.PHONY: semgrep
semgrep:
	@$(semgrep) --quiet --metrics=off --config="r/dgryski.semgrep-go" .

.PHONY: check-fmt
check-fmt:
	@if [ $$(go fmt -mod=vendor ./...) ]; then\
		echo "Go code is not formatted";\
		exit 1;\
	fi

.PHONY: check-codegen
check-codegen: gogenerate
	@git diff --exit-code --

.PHONY: check-gomod
check-gomod:
	@go mod tidy
	@git diff --exit-code -- go.sum go.mod

.PHONY: gogenerate
gogenerate:
	@go generate -mod vendor ./...