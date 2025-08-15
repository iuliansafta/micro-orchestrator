package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/iuliansafta/micro-orchestrator/api/proto"
	"github.com/iuliansafta/micro-orchestrator/pkg/api"
	"github.com/iuliansafta/micro-orchestrator/pkg/health"
	"github.com/iuliansafta/micro-orchestrator/pkg/scheduler"
	"github.com/iuliansafta/micro-orchestrator/pkg/types"
	"google.golang.org/grpc"
)

var (
	grpcPort = flag.String("grpc-port", "50051", "gRPC service port")
	strategy = flag.String("strategy", "binpack", "Scheduling strategy (binpack, spread, random)")
)

func main() {
	flag.Parse()

	// Init scheduler
	sched := scheduler.NewScheduler(scheduler.Strategy(*strategy))

	// Init health monitor
	healthMonitor := health.NewHealthMonitor()
	registerTestNodes(sched)

	// Init API service
	apiServer := api.NewServer(sched, healthMonitor)

	// Create listener
	listener, err := net.Listen("tcp", ":"+*grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create the gRPC service
	grpcServer := grpc.NewServer()
	pb.RegisterOrchestratorServer(grpcServer, apiServer)

	// Start healtchecks
	ctx, cancel := context.WithCancel(context.Background())
	go healthMonitor.Start(ctx)

	// Start gRPC server
	go func() {
		log.Printf("Starting gRPC server on :%s", *grpcPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	cancel()
	grpcServer.GracefulStop()
}

func registerTestNodes(schd *scheduler.Scheduler) {
	regions := []string{"us-east-1", "eu-west-1", "ap-southest-1"}

	for i, region := range regions {
		for j := range 3 {
			node := &types.Node{
				ID:       fmt.Sprintf("node-%s-%d", region, j),
				Region:   region,
				TotalCPU: 8.0,
				TotalMem: 2 * 16384, // 16GB
				Healthy:  true,
				LastSeen: time.Now(),
			}

			// add latency for regions simulation
			if i > 0 {
				time.Sleep(time.Duration(i*50) * time.Millisecond)
			}

			if err := schd.RegisterNode(node); err != nil {
				log.Printf("failed to register node: %v\n", err)
			}
		}
	}
}

func testAssingContainers(schd *scheduler.Scheduler, hm *health.HealthMonitor) {
	for i := range 2 {
		container := &types.Container{
			ID:     fmt.Sprintf("%d-%d", i, i),
			Name:   fmt.Sprintf("%d-%d", i, i),
			Image:  "ubuntu-1",
			CPU:    float64(i) * 2.0,
			Memory: int64(i) * 16384,
			Region: "us-east-1",
			Labels: nil,
			State:  types.ContainerRunning,
			RestartPolicy: types.RestartPolicy{
				Type:       "on-failure",
				MaxRetries: 2,
				Backoff:    10,
			},
			CreatedAt: time.Now(),
			HealthCheck: &types.HealthCheck{
				Type:     "http",
				Endpoint: "/health",
				Retries:  2,
			},
		}

		node, err := schd.Schedule(container)

		if err != nil {
			log.Fatalf("failed to schedule the container: %v", err)
		}

		container.NodeID = node.ID
		container.Region = node.Region
		container.State = types.ContainerRunning

		hm.RegisterContainer(container)
	}
}
