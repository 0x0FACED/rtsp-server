.PHONY: all
all: run

.PHONY: run
run:
	@echo "Running application..."
	@go run ./cmd/server/main/main.go

