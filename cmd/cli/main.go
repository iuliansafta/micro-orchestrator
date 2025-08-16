package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/iuliansafta/micro-orchestrator/api/proto"
)

func main() {
	var (
		server   = flag.String("server", "localhost:50051", "Server address")
		action   = flag.String("action", "deploy", "Action: deploy, scale, delete, status")
		name     = flag.String("name", "test-app", "Deployment name")
		image    = flag.String("image", "nginx:latest", "Container image")
		replicas = flag.Int("replicas", 3, "Number of replicas")
		cpu      = flag.Float64("cpu", 0.5, "CPU cores")
		memory   = flag.Int64("memory", 512, "Memory in MB")
		region   = flag.String("region", "", "Target region")
	)
	flag.Parse()

	// Connect to server
	conn, err := grpc.NewClient(*server, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewOrchestratorClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch *action {
	case "deploy":
		resp, err := client.Deploy(ctx, &pb.DeployRequest{
			Name:     *name,
			Image:    *image,
			Replicas: int32(*replicas),
			Cpu:      *cpu,
			Memory:   *memory,
			Region:   *region,
			HealthCheck: &pb.HealthCheck{
				Type:            "http",
				Endpoint:        "/health",
				IntervalSeconds: 10,
				TimeoutSeconds:  5,
				Retries:         3,
			},
			Strategy: &pb.DeploymentStrategy{
				Type:           "rolling",
				MaxUnavailable: 1,
				MaxSurge:       1,
			},
		})

		if err != nil {
			log.Fatalf("Deployment failed: %v", err)
		}

		fmt.Printf("Deployment successful!\n")
		fmt.Printf("ID: %s\n", resp.DeploymentId)
		fmt.Printf("Status: %s\n", resp.Status)
		fmt.Printf("Containers: %v\n", resp.ContainerIds)
	}
}
