.PHONY: build test run clean proto docker-build docker-up docker-down docker-logs docker-clean

build:
	go build -o bin/orchestrator cmd/orchestrator/main.go
	go build -o bin/cli cmd/cli/main.go

install-tools:
	@echo "Installing protoc-gen-go..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@echo "Installing protoc-gen-go-grpc..."
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Tools installed!"

check-tools:
	@which protoc > /dev/null || (echo "protoc not found. Please install protobuf compiler" && exit 1)
	@which protoc-gen-go > /dev/null || (echo "protoc-gen-go not found. Run 'make install-tools'" && exit 1)
	@which protoc-gen-go-grpc > /dev/null || (echo "protoc-gen-go-grpc not found. Run 'make install-tools'" && exit 1)
	@echo "All tools found!"

proto:
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
	        --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	        api/proto/orchestrator.proto
	@echo "Done!"

run:
	docker-compose up -d

clean:
	@rm -rf bin/
	@rm -f api/proto/*.pb.go
	docker-compose down

fmt:
	go fmt ./...
	goimports -w .

lint:
	golangci-lint run


# DOCKER
docker-build:
	@echo "Building Docker images..."
	@docker-compose build

docker-up: docker-build
	@echo "Starting services..."
	@docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Services are up!"
	@echo "Orchestrator API: http://localhost:50051"
	@echo "Metrics: http://localhost:8080/metrics"
	@echo "Prometheus: http://localhost:9090"
	@echo "Grafana: http://localhost:3000 (admin/admin)"

docker-down:
	@echo "Stopping services..."
	@docker-compose down

docker-logs:
		@docker-compose logs -f

docker-clean:
		@echo "Cleaning up..."
		@docker-compose down -v
		@docker system prune -f

demo: docker-up
	@echo "Waiting for orchestrator to be ready..."
	@sleep 10
	@echo "Deploying demo application..."
	@docker-compose exec orchestrator /app/cli \
		-server=localhost:50051 \
		-action=deploy \
		-name=demo-app \
		-image=nginx:latest \
		-replicas=10 \
		-cpu=0.5 \
		-memory=512 \
		-region=us-east-1
	@echo "Checking metrics..."
	@sleep 5
	@docker-compose exec orchestrator /app/cli \
		-server=localhost:50051 \
		-action=metrics
