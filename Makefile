.PHONY: check fmt vet lint generate test build proto install installctl dockerimage

generate:
	@echo "Generating mock files..."
	@bash scripts/generate.sh

fmt:
	@echo "Checking Go file formatting..."
	@bash scripts/checks/fmt.sh

vet:
	@echo "Checking for common Go mistakes..."
	@bash scripts/checks/vet.sh

lint:
	@echo "Checking for style errors..."
	@bash scripts/checks/lint.sh

check: generate fmt vet lint

test: generate
	@echo "Running tests..."
	@bash scripts/test.sh

build:
	@echo "Compiling jackal binary..."
	@bash scripts/compile.sh

proto:
	@echo "Generating proto files..."
	@command -v protoc >/dev/null 2>&1 || { echo 'Please install protoc or use image that has it'; exit 1; }
	@protoc --proto_path="${GOPATH}"/src --proto_path=. --go_out=. proto/model/*.proto

install:
	@echo "Installing jackal binary..."
	@bash scripts/install.sh

installctl:
	@echo "Installing jackalctl binary..."
	@bash scripts/installctl.sh

dockerimage:
	@echo "Building docker image..."
	@bash scripts/dockerimage.sh
