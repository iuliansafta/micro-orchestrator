.PHONY: build test run clean proto

build:
	go build -o bin/orchestrator cmd/orchestrator/main.go

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

demo:
	@echo "Starting orchestrator..."
	@docker-compose up -d
	@sleep 5
	@echo "Deploying test application..."
	@./bin/cli -action=deploy -name=demo-app -replicas=10
	@sleep 2
	@echo "Checking metrics..."
	@./bin/cli -action=metrics
