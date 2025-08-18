package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/iuliansafta/micro-orchestrator/api/proto"
	"github.com/iuliansafta/micro-orchestrator/pkg/api"
	"github.com/iuliansafta/micro-orchestrator/pkg/health"
	"github.com/iuliansafta/micro-orchestrator/pkg/metrics"
	"github.com/iuliansafta/micro-orchestrator/pkg/scheduler"
	"github.com/iuliansafta/micro-orchestrator/pkg/types"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

var (
	grpcPort    = flag.String("grpc-port", "50051", "gRPC service port")
	metricsPort = flag.String("metrics-port", "8080", "Metrics server port")
	strategy    = flag.String("strategy", "binpack", "Scheduling strategy (binpack, spread)")
)

func main() {
	flag.Parse()

	// Init metric collector
	metricsCollector := metrics.NewMetricsCollector()

	// Init scheduler
	sched := scheduler.NewScheduler(scheduler.Strategy(*strategy), metricsCollector)

	// Init health monitor
	healthMonitor := health.NewHealthMonitor(metricsCollector)
	registerNodes(sched, metricsCollector)

	// Init API service
	apiServer := api.NewServer(sched, healthMonitor, metricsCollector)

	// Create listener
	listener, err := net.Listen("tcp", ":"+*grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create the gRPC service
	grpcServer := grpc.NewServer()
	pb.RegisterOrchestratorServer(grpcServer, apiServer)

	// Metrics
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Printf("Starting metrics server on :%s", *metricsPort)
		if err := http.ListenAndServe(":"+*metricsPort, nil); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

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

func registerNodes(schd *scheduler.Scheduler, mc *metrics.MetricsCollector) {
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

			mc.NodeRegistered(region)
		}
	}
}
