.PHONY: run build clean tidy ingest_data

APP_NAME=jedi-team-challenge
BINARY_NAME=$(APP_NAME)
CMD_PATH=./cmd/server

# Default target
all: build

run: build
	@echo ">>> Running application..."
	./$(BINARY_NAME)

# Run with data ingestion first (if DB is empty or data.md changed)
run-fresh: clean_db ingest_data run

build: tidy
	@echo ">>> Building application..."
	@go build -o $(BINARY_NAME) $(CMD_PATH)/main.go

# Command to ingest data.md into the database
ingest_data:
	@echo ">>> Ingesting data from data.md..."
	@go run $(CMD_PATH)/main.go -ingest

# Clean build artifacts
clean:
	@echo ">>> Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)

# Clean database
clean_db:
	@echo ">>> Cleaning database..."
	@rm -f gwi_chatbot.db # Assuming default DB name

# Tidy dependencies
tidy:
	@echo ">>> Tidying dependencies..."
	@go mod tidy

help:
	@echo "Available targets:"
	@echo "  all           - Build the application (default)"
	@echo "  run           - Build and run the application"
	@echo "  run-fresh     - Clean database, ingest data, then build and run"
	@echo "  build         - Build the application binary"
	@echo "  ingest_data   - Run data ingestion script"
	@echo "  clean         - Remove build artifacts"
	@echo "  clean_db      - Remove the SQLite database file"
	@echo "  tidy          - Tidy Go module dependencies"