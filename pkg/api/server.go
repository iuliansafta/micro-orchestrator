package api

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/iuliansafta/micro-orchestrator/api/proto"
	"github.com/iuliansafta/micro-orchestrator/pkg/health"
	"github.com/iuliansafta/micro-orchestrator/pkg/metrics"
	"github.com/iuliansafta/micro-orchestrator/pkg/scheduler"
	"github.com/iuliansafta/micro-orchestrator/pkg/types"
)

type Server struct {
	pb.UnimplementedOrchestratorServer
	mu           sync.Mutex
	scheduler    *scheduler.Scheduler
	deployments  map[string]*types.Deployments
	eventStreams map[string]chan *pb.Event
	hm           *health.HealthMonitor
	mc           *metrics.MetricsCollector
}

func NewServer(sched *scheduler.Scheduler, hm *health.HealthMonitor, mc *metrics.MetricsCollector) *Server {
	return &Server{
		scheduler:    sched,
		deployments:  make(map[string]*types.Deployments),
		eventStreams: make(map[string]chan *pb.Event),
		hm:           hm,
		mc:           mc,
	}
}

func (s *Server) Deploy(ctx context.Context, req *pb.DeployRequest) (*pb.DeployResponse, error) {
	deploymentStart := time.Now()
	deploymentID := generteID()

	deployment := &types.Deployments{
		ID:        deploymentID,
		Name:      req.Name,
		Status:    "DEPLOYING",
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.deployments[deployment.ID] = deployment
	s.mu.Unlock()

	// Create containers based on the replica
	containerIDs := []string{}
	successCount := 0

	for i := 0; i < int(req.Replicas); i++ {
		container := &types.Container{
			ID:     fmt.Sprintf("%s-%d", deploymentID, i),
			Name:   fmt.Sprintf("%s-%d", req.Name, i),
			Image:  req.Image,
			CPU:    req.Cpu,
			Memory: req.Memory,
			Region: req.Region,
			Labels: req.Labels,
			State:  types.ContainerPending,
			RestartPolicy: types.RestartPolicy{
				Type:       "on-failure",
				MaxRetries: 2,
				Backoff:    10,
			},
			CreatedAt: time.Now(),
		}

		if req.HealthCheck != nil {
			container.HealthCheck = &types.HealthCheck{
				Type:     req.HealthCheck.Type,
				Endpoint: req.HealthCheck.Endpoint,
				Interval: time.Duration(req.HealthCheck.IntervalSeconds) * time.Second,
				Timeout:  time.Duration(req.HealthCheck.TimeoutSeconds) * time.Second,
				Retries:  int(req.HealthCheck.Retries),
			}
		}

		node, err := s.scheduler.Schedule(container)

		if err != nil {
			log.Printf("SCHEDULING_FAILED: %v", err)
			s.mc.SchedulingFailed(err.Error())
			continue
		}

		container.NodeID = node.ID
		container.State = types.ContainerRunning
		containerIDs = append(containerIDs, container.ID)
		successCount++

		// Register container for health checks
		s.hm.RegisterContainer(container)
	}

	deployment.SuccessRate = float64(successCount) / float64(req.Replicas) * 100

	if deployment.SuccessRate >= 95 {
		deployment.Status = "SUCCESS"
	} else if deployment.SuccessRate > 0 {
		deployment.Status = "PARTIAL_SUCCESS"
	} else {
		deployment.Status = "FAILED"
	}

	duration := time.Since(deploymentStart)
	s.mc.DeploymentCompleted(deployment.Status, deployment.SuccessRate)
	s.mc.DeploymentDuration(deployment.Status, duration)

	return &pb.DeployResponse{
		DeploymentId: deploymentID,
		Status:       deployment.Status,
		ContainerIds: containerIDs,
	}, nil
}

func (s *Server) GetMetrics(ctx context.Context, req *pb.MetricsRequest) (*pb.MetricsResponse, error) {
	metrics := s.mc.GetCurrentMetrics()

	regionMetrics := make(map[string]*pb.RegionMetrics)

	for region, rm := range metrics.RegionMetrics {
		regionMetrics[region] = &pb.RegionMetrics{
			ContainerCount:    rm.ContainerCount,
			CpuUtilization:    rm.CPUUtilization,
			MemoryUtilization: rm.MemoryUtilization,
			Availability:      rm.Availability,
		}
	}

	return &pb.MetricsResponse{
		DeploymentSuccessRate: metrics.DeploymentSuccessRate,
		TotalContainers:       int32(metrics.TotalContainers),
		HealthyContainers:     int32(metrics.HealthyContainers),
		RegionMetrics:         regionMetrics,
	}, nil
}

func generteID() string {
	return uuid.NewString()
}
