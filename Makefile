.POSIX:
.SUFFIXES:

GOFILES!=find . -name '*.go'

GOLDFLAGS =-s -w -extldflags $(LDFLAGS)

.PHONY: install
install:
	@go install -ldflags="-s -w" github.com/ortuman/jackal

.PHONY: install-tools
install-tools:
	@env GO111MODULE=off go get -u \
		golang.org/x/lint/golint \
		golang.org/x/tools/cmd/goimports

.PHONY: fmt
fmt: install-tools
	@echo "Checking go files format..."
	@GOIMP=$$(for f in $$(find . -type f -name "*.go" ! -path "./.cache/*" ! -path "./vendor/*" ! -name "bindata.go") ; do \
		goimports -l $$f ; \
		done) && echo $$GOIMP && test -z "$$GOIMP"

go.sum: $(GOFILES) go.mod
	go mod tidy

jackal: $(GOFILES) go.mod go.sum
	@echo "Building binary..."
	@go build\
		-trimpath \
		-o $@ \
		-ldflags "$(GOLDFLAGS)"

.PHONY: build
build: jackal

.PHONY: test
test:
	@echo "Running tests..."
	@go test -race $$(go list ./...)

.PHONY: coverate
coverage:
	@echo "Generating coverage profile..."
	@go test -race -coverprofile=coverage.txt -covermode=atomic $$(go list ./...)

.PHONY: vet
vet:
	@echo "Searching for buggy code..."
	@go vet $$(go list ./...)

.PHONY: lint
lint: install-tools
	@echo "Running linter..."
	@golint $$(go list ./...)

.PHONY: dockerimage
dockerimage:
	@echo "Building binary..."
	@env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w"
	@echo "Building docker image..."
	@docker build -f dockerfiles/Dockerfile -t ortuman/jackal .

.PHONY: clean
clean:
	@go clean
