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
	mu       sync.RWMutex
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	eligibleNodes := s.filterNodes(container)

	if len(eligibleNodes) == 0 {
		return nil, fmt.Errorf("no eligible nodes found")
	}

	var selectedNode *types.Node

	switch s.strategy {
	case StrategyBinPack:
		selectedNode = s.binPackStrategy(eligibleNodes)

		// TBD with the rest of the strategies
	}

	if selectedNode == nil {
		return nil, fmt.Errorf("insuficient resources on all nodes")
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

func (s *Scheduler) binPackStrategy(nodes []*types.Node) *types.Node {
	// eligibleNodes := make([]*types.Node, 0)

	// for _, node := range nodes {
	// 	availbeCPU := node.TotalCPU - node.UsedCPU
	// 	availabeMem := node.TotalMem - node.UsedMem

	// 	if availbeCPU >= container.CPU && availabeMem >= container.Memory {
	// 		eligibleNodes = append(eligibleNodes, node)
	// 	}
	// }

	// if len(eligibleNodes) == 0 {
	// 	return nil
	// }

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
