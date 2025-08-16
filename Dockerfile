FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git make protobuf protobuf-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

COPY . .

RUN protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    api/proto/orchestrator.proto

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o orchestrator cmd/orchestrator/main.go && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cli cmd/cli/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 1000 orchestrator && \
    adduser -D -u 1000 -G orchestrator orchestrator

WORKDIR /app

COPY --from=builder /app/orchestrator /app/orchestrator
COPY --from=builder /app/cli /app/cli

RUN chown -R orchestrator:orchestrator /app

USER orchestrator

EXPOSE 50051 8080

CMD ["/app/orchestrator"]
