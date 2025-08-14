.PHONY: build test run clean proto

build:
	go build -o bin/orchestrator cmd/orchestrator/main.go

run:
	docker-compose up -d

clean:
	rm -rf bin/
	docker-compose down

fmt:
	go fmt ./...
	goimports -w .

lint:
	golangci-lint run

demo:
	@echo "Starting orchestrator..."
	@docker-compose up -d
	@sleep 5
	@echo "Deploying test application..."
	@./bin/cli -action=deploy -name=demo-app -replicas=10
	@sleep 2
	@echo "Checking metrics..."
	@./bin/cli -action=metrics
