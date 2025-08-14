package scheduler

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/iuliansafta/micro-orchestrator/pkg/types"
)

type Strategy string

const (
	StrategyBinPack Strategy = "binpack" // maximize resource utilization by packing containers onto nodes as tightly as possible
	StrategySpread  Strategy = "spread"  // distribute across nodes
	StrategyRandom  Strategy = "random"
)

type Scheduler struct {
	mu       sync.Mutex
	nodes    map[string]*types.Node
	strategy Strategy
}

func NewScheduler(strategy Strategy) *Scheduler {
	return &Scheduler{
		nodes:    make(map[string]*types.Node),
		strategy: strategy,
	}
}

func (s *Scheduler) RegisterNode(node *types.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nodes[node.ID] = node

	return nil
}

func (s *Scheduler) Schedule(container *types.Container) (*types.Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	eligibleNodes := s.filterNodes(container)

	if len(eligibleNodes) == 0 {
		return nil, fmt.Errorf("no eligible nodes found")
	}

	var selectedNode *types.Node

	switch s.strategy {
	case StrategyBinPack:
		selectedNode = s.binPackStrategy(eligibleNodes)
	case StrategySpread:
		selectedNode = s.spreadStrategy(eligibleNodes)
	}

	if selectedNode == nil {
		return nil, fmt.Errorf("insufficient resources on all nodes")
	}

	// update node resoruces
	selectedNode.UsedCPU += container.CPU
	selectedNode.UsedMem += container.Memory
	selectedNode.Containers = append(selectedNode.Containers, container.ID)

	log.Print("Scheduling success \n")
	log.Printf("Container %+v \n", container)
	log.Printf("On node %+v \n", selectedNode)

	return selectedNode, nil
}

func (s *Scheduler) filterNodes(container *types.Container) []*types.Node {
	var nodes []*types.Node

	for _, node := range s.nodes {
		if !node.Healthy {
			continue
		}

		// check if node is outdated (not seen in the last 30 sec)
		if time.Since(node.LastSeen) > 30*time.Second {
			continue
		}

		availableCPU := node.TotalCPU - node.UsedCPU
		availableMem := node.TotalMem - node.UsedMem

		if availableCPU >= container.CPU && availableMem >= container.Memory {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

// binPackStrateg
func (s *Scheduler) binPackStrategy(nodes []*types.Node) *types.Node {
	// Sort nodes by utilization
	sort.Slice(nodes, func(i, j int) bool {
		// calculate utilization for the i node
		cpuUtilizationI := nodes[i].UsedCPU / nodes[i].TotalCPU
		memUtilizationI := float64(nodes[i].UsedMem) / float64(nodes[i].TotalMem)

		// AVG of CPU and Mem utlization
		utilizationI := (cpuUtilizationI + memUtilizationI) / 2

		// calculate utilization for the j node
		cpuUtilizationJ := nodes[j].UsedCPU / nodes[j].TotalCPU
		memUtilizationJ := float64(nodes[j].UsedMem) / float64(nodes[j].TotalMem)

		// AVG
		utilizationJ := (cpuUtilizationJ + memUtilizationJ) / 2

		return utilizationI > utilizationJ
	})

	// let's return the first node (most utilized one)
	if len(nodes) > 0 {
		return nodes[0]
	}

	return nil
}

// spreadStrategy least utilized first
func (s *Scheduler) spreadStrategy(nodes []*types.Node) *types.Node {
	sort.Slice(nodes, func(i int, j int) bool {
		return len(nodes[i].Containers) < len(nodes[j].Containers)
	})

	if len(nodes) > 0 {
		return nodes[0]
	}

	return nil
}
