install:
	@go install -ldflags="-s -w" github.com/ortuman/jackal

install-tools:
	@go get -u \
		golang.org/x/lint/golint \
		golang.org/x/tools/cmd/goimports

fmt: install-tools
	@echo "Checking go files format..."
	@GOIMP=$$(for f in $$(find . -type f -name "*.go" ! -path "./.cache/*" ! -path "./vendor/*" ! -name "bindata.go") ; do \
    		goimports -l $$f ; \
    	done) && echo $$GOIMP && test -z "$$GOIMP"

build:
	@echo "Building binary..."
	@go build -ldflags="-s -w"

test:
	@echo "Running tests..."
	@go test -race $$(go list ./...)

coverage:
	@echo "Generating coverage profile..."
	@go test -race -coverprofile=coverage.txt -covermode=atomic $$(go list ./...)

vet:
	@echo "Searching for buggy code..."
	@go vet $$(go list ./...)

lint: install-tools
	@echo "Running linter..."
	@golint $$(go list ./...)

dockerimage:
	@echo "Building binary..."
	@env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w"
	@echo "Building docker image..."
	@docker build -f dockerfiles/Dockerfile -t ortuman/jackal .

clean:
	@go clean
