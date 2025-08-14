package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iuliansafta/micro-orchestrator/pkg/scheduler"
	"github.com/iuliansafta/micro-orchestrator/pkg/types"
)

var (
	strategy = flag.String("strategy", "binpack", "Scheduling strategy (binpack, spread, random)")
)

func main() {
	flag.Parse()

	// Init scheduler
	sched := scheduler.NewScheduler(scheduler.Strategy(*strategy))

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

			if err := sched.RegisterNode(node); err != nil {
				log.Printf("failed to register node: %v\n", err)
			}
		}
	}

	container := &types.Container{
		ID:        fmt.Sprintf("%s-%d", "1", 1),
		Name:      fmt.Sprintf("%s-%d", "MyApp", 1),
		Image:     "ubuntu-1",
		CPU:       2.0,
		Memory:    16384,
		Region:    "us-east-1",
		Labels:    nil,
		State:     types.ContainerPending,
		CreatedAt: time.Now(),
	}

	node, err := sched.Schedule(container)

	if err != nil {
		log.Fatalf("failed to schedule the container: %v", err)
	}

	container.NodeID = node.ID
	container.Region = node.Region
	container.State = types.ContainerRunning

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}
