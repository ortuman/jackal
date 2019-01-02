install:
	@export GO111MODULE=on && go install -ldflags="-s -w" github.com/ortuman/jackal

install-tools:
	@export GO111MODULE=on && go get -u \
		golang.org/x/lint/golint

test:
	@go test -race $$(go list ./...)

coverage:
	@go test -race -coverprofile=coverage.txt -covermode=atomic $$(go list ./...)

vet:
	@go vet $$(go list ./...)

lint: install-tools
	@golint $$(go list ./...)

clean:
	@go clean
