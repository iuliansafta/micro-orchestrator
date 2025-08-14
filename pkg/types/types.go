package types

import "time"

type ContainerState string

const (
	ContainerPending  ContainerState = "pending"
	ContainerRunning  ContainerState = "running"
	ContainerFailed   ContainerState = "failed"
	CntainerCompleted ContainerState = "completed"
)

type Container struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Image         string            `json:"image"`
	State         ContainerState    `json:"state"`
	NodeID        string            `json:"node_id"`
	Region        string            `json:"region"`
	CPU           float64           `json:"cpu"`    // CPU cores
	Memory        int64             `json:"memory"` // Memory in MB
	Replicas      int               `json:"replicas"`
	Labels        map[string]string `json:"labels"`
	HealthCheck   *HealthCheck      `json:"health_check,omitempty"`
	RestartPolicy RestartPolicy     `json:"restart_policy"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type Node struct {
	ID         string    `json:"id"`
	Region     string    `json:"region"`
	TotalCPU   float64   `json:"total_cpu"`
	TotalMem   int64     `json:"total_memory"`
	UsedCPU    float64   `json:"used_cpu"`
	UsedMem    int64     `json:"used_memory"`
	Containers []string  `json:"containers"`
	Healthy    bool      `json:"healthy"`
	LastSeen   time.Time `json:"last_seen"`
}

type HealthCheck struct {
	Type             string        `json:"type"` // http, tcp, exec
	Endpoint         string        `json:"endpoint"`
	Interval         time.Duration `json:"interval"`
	Timeout          time.Duration `json:"timeout"`
	Retries          int           `json:"retries"`
	ConsecutiveFails int           `json:"consecutive_fails"`
	LastCheck        time.Time     `json:"last_check"`
	Healthy          bool          `json:"healthy"`
}

type RestartPolicy struct {
	Type       string `json:"type"` // always, on-failure, never
	MaxRetries int    `json:"max_retries"`
	Backoff    int    `json:"backoff_seconds"`
}
