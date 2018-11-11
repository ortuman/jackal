install:
	@export GO111MODULE=on && go install github.com/ortuman/jackal

test:
	@echo "Running tests..."
	@go test -race $$(go list ./...)

coverage:
	@go test -race -coverprofile=coverage.txt -covermode=atomic $$(go list ./...)

vet:
	@echo "Looking for buggy code..."
	@go vet $$(go list ./...)

clean:
	@go clean
