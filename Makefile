install:
	@go install github.com/ortuman/jackal

test:
	@echo "Running tests..."
	@go test $$(go list ./...)

vet:
	@echo "Looking for buggy code..."
	@go vet $$(go list ./...)

clean:
	@go clean
