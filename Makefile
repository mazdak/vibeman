.PHONY: build swagger generate-api dev clean

# Default target
all: swagger build

# Generate Swagger/OpenAPI documentation
swagger:
	@echo "Generating OpenAPI spec..."
	@$$(go env GOPATH)/bin/swag init -g main.go -o docs

# Build the Go binary
build: swagger
	@echo "Building vibeman..."
	@go build -o vibeman .

# Generate frontend API client from OpenAPI spec
generate-api: swagger
	@echo "Generating TypeScript API client..."
	@cd vibeman-web && bun run generate-api

# Development workflow - regenerate everything
dev: swagger generate-api
	@echo "Starting development server..."
	@go run .

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	@rm -rf docs/
	@rm -rf vibeman-web/src/generated/api/
	@rm -f vibeman

# Install swag if not present
install-swag:
	@echo "Installing swag..."
	@go install github.com/swaggo/swag/cmd/swag@latest

# Watch mode for development
watch:
	@echo "Starting watch mode..."
	@find . -name "*.go" -not -path "./vendor/*" -not -path "./docs/*" | entr -r make dev