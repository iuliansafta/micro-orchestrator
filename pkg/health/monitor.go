package health

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/iuliansafta/micro-orchestrator/pkg/types"
)

type HealthMonitor struct {
	mu            sync.RWMutex
	containers    map[string]*types.Container
	checkInterval time.Duration
	restartChan   chan string
	registerChan  chan *types.Container
}

func NewHealthMonitor() *HealthMonitor {
	return &HealthMonitor{
		containers:    make(map[string]*types.Container),
		checkInterval: 10 * time.Second,
		restartChan:   make(chan string, 100),
		registerChan:  make(chan *types.Container, 100),
	}
}

func (h *HealthMonitor) Start(ctx context.Context) {
	go h.handleContainerUpdates(ctx)

	go h.healthCheckLoop(ctx)

	go h.handleRestarts(ctx)
}

func (h *HealthMonitor) RegisterContainer(container *types.Container) {
	if container.HealthCheck == nil {
		return
	}

	// write container to the channel
	h.registerChan <- container

	log.Printf("HealthMonitor: Registered container %s for health monitoring\n", container.ID)
}

func (h *HealthMonitor) handleContainerUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case container := <-h.registerChan:
			h.mu.Lock()
			h.containers[container.ID] = container
			h.mu.Unlock()

			// Start health check for this container
			h.checkContainerHealth(container)
		}
		// case container := <-h.restartChan
	}
}

func (h *HealthMonitor) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.performHealthChecks()
		}
	}
}

func (h *HealthMonitor) performHealthChecks() {
	h.mu.RLock()
	// mem eficienty
	containers := make([]*types.Container, 0, len(h.containers))

	for _, c := range h.containers {
		if c.HealthCheck != nil && c.State == types.ContainerRunning {
			containers = append(containers, c)
		}
	}
	h.mu.RUnlock()

	var wg sync.WaitGroup
	for _, container := range containers {
		wg.Add(1)
		go func(c *types.Container) {
			defer wg.Done()
			h.checkContainerHealth(c)
		}(container)
	}

	wg.Wait()
}

func (h *HealthMonitor) checkContainerHealth(container *types.Container) {
	hc := container.HealthCheck

	healthy := false

	switch hc.Type {
	case "http":
		healthy = h.performHTTPCheck(container, hc)
	}

	hc.LastCheck = time.Now()

	if healthy {
		hc.ConsecutiveFails = 0
		hc.Healthy = true

		//TODO: update metrics healthceck success for container ID
		log.Printf("Healthceck success container ID: %s", container.ID)
	} else {
		hc.ConsecutiveFails++

		//TODO: healthceck failed container ID
		log.Printf("Healthceck failed container ID: %s ==> %v", container.ID, hc.ConsecutiveFails >= hc.Retries)
		if hc.ConsecutiveFails >= hc.Retries {
			hc.Healthy = false
			container.State = types.ContainerFailed
			h.handleUnhealtyContainer(container)
		}
	}
}

func (h *HealthMonitor) performHTTPCheck(container *types.Container, hc *types.HealthCheck) bool {
	client := &http.Client{
		Timeout: hc.Timeout,
	}

	url := fmt.Sprintf("http://%s:%s%s", container.NodeID, "80", hc.Endpoint)

	resp, err := client.Get(url)
	if err != nil {
		return false
	}

	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (h *HealthMonitor) handleUnhealtyContainer(container *types.Container) {
	policy := container.RestartPolicy

	switch policy.Type {
	case "always":
		h.scheduleRestart(container, 0)
	case "on-failure":
		if container.State == types.ContainerFailed {
			backoff := time.Duration(policy.Backoff) * time.Second
			h.scheduleRestart(container, backoff)
		}
	case "never":

	}
}

func (h *HealthMonitor) scheduleRestart(container *types.Container, backoff time.Duration) {
	go func() {
		if backoff > 0 {
			time.Sleep(backoff)
		}
		h.restartChan <- container.ID
		//TODO: Metrics container restarted
		log.Printf("Restarted container: %s", container.ID)
	}()
}

func (h *HealthMonitor) handleRestarts(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case containerID := <-h.restartChan:
			h.mu.Lock()
			container, exist := h.containers[containerID]
			h.mu.Unlock()

			if !exist {
				log.Printf("Container %s not found", containerID)
				return
			}

			if err := h.performContainerRestart(container); err != nil {
				log.Printf("Failed to restart the container %s", containerID)

				h.mu.Lock()
				container.State = types.ContainerRunning
				h.mu.Unlock()
			} else {
				log.Printf("Successfully restarted container: %s", containerID)

				h.mu.Lock()
				container.State = types.ContainerRunning
				h.mu.Unlock()
			}

		}
	}
}

func (h *HealthMonitor) performContainerRestart(container *types.Container) error {
	if container.HealthCheck != nil {
		container.HealthCheck.ConsecutiveFails = 0
		container.HealthCheck.Healthy = true
		container.HealthCheck.LastCheck = time.Time{}
	}

	return nil
}
