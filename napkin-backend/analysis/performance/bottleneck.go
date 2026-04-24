package performance

import (
	"fmt"
	"math"
	"napkin-backend/graph"
	"sort"
)

type Bottleneck struct {
	Type         string // node or edge
	ID           string
	IncomingFlow float64
	Capacity     float64
	Drop         float64
}

func min(vals ...float64) float64 {
	m := vals[0]
	for _, v := range vals {
		if v < m {
			m = v
		}
	}
	return m
}

func getReplicaCount(n *int) float64 {
	if n == nil {
		return 1
	}
	return float64(*n)
}

func findBottlenecksFromIntentGraph(nodes map[string]Node, edges []Edge, nodeThroughput map[string]float64) []Bottleneck {
	var bottlenecks []Bottleneck

	for _, node := range nodes {
		if node.IsPassthrough {
			continue
		}

		incomingEdges := getIncomingEdgesFromList(node.ID, edges)
		if len(incomingEdges) == 0 {
			continue
		}

		incomingTotal := 0.0
		for _, edge := range incomingEdges {
			upstream := nodeThroughput[edge.SourceNode]

			if edge.BandwidthLimit != nil {
				if upstream > *edge.BandwidthLimit {
					bottlenecks = append(bottlenecks, Bottleneck{
						Type:         "edge",
						ID:           edge.SourceNode + "-" + edge.TargetNode,
						IncomingFlow: upstream,
						Capacity:     *edge.BandwidthLimit,
						Drop:         upstream - *edge.BandwidthLimit,
					})
				}
				upstream = math.Min(upstream, *edge.BandwidthLimit)
			}
			incomingTotal += upstream
		}

		switch node.Type {
		case NodeTypeEC2:
			effectiveCapacity := node.MaxThroughput * getReplicaCount(node.ReplicaCount)
			if incomingTotal > effectiveCapacity {
				bottlenecks = append(bottlenecks, Bottleneck{
					Type:         "node",
					ID:           node.ID,
					IncomingFlow: incomingTotal,
					Capacity:     effectiveCapacity,
					Drop:         incomingTotal - effectiveCapacity,
				})
			}

		case NodeTypeDatabase:
			effectiveCapacity := node.MaxThroughput * getReplicaCount(node.ReplicaCount)
			if node.MaxReadThroughput != nil || node.MaxWriteThroughput != nil {
				read := math.Inf(1)
				write := math.Inf(1)
				if node.MaxReadThroughput != nil {
					read = *node.MaxReadThroughput
				}
				if node.MaxWriteThroughput != nil {
					write = *node.MaxWriteThroughput
				}
				effectiveCapacity = min(effectiveCapacity, read, write)
			}
			if incomingTotal > effectiveCapacity {
				bottlenecks = append(bottlenecks, Bottleneck{
					Type:         "node",
					ID:           node.ID,
					IncomingFlow: incomingTotal,
					Capacity:     effectiveCapacity,
					Drop:         incomingTotal - effectiveCapacity,
				})
			}

		case NodeTypeStorageBucket:
			effectiveCapacity := node.MaxThroughput
			if node.MaxReadThroughput != nil || node.MaxWriteThroughput != nil {
				read := math.Inf(1)
				write := math.Inf(1)
				if node.MaxReadThroughput != nil {
					read = *node.MaxReadThroughput
				}
				if node.MaxWriteThroughput != nil {
					write = *node.MaxWriteThroughput
				}
				effectiveCapacity = min(effectiveCapacity, read, write)
			}
			if incomingTotal > effectiveCapacity {
				bottlenecks = append(bottlenecks, Bottleneck{
					Type:         "node",
					ID:           node.ID,
					IncomingFlow: incomingTotal,
					Capacity:     effectiveCapacity,
					Drop:         incomingTotal - effectiveCapacity,
				})
			}
		}
	}

	sort.Slice(bottlenecks, func(i, j int) bool {
		return bottlenecks[i].Drop > bottlenecks[j].Drop
	})

	return bottlenecks
}

func FindBottlenecks(intentGraph *graph.DirectedGraph) ([]Bottleneck, error) {
	perfNodes, err := buildIntentGraphPerformanceNodes(intentGraph)
	if err != nil {
		return nil, err
	}

	edges := getIntentGraphEdges(intentGraph)

	orderedIDs, err := topologicalSortIntentGraph(perfNodes, edges)
	if err != nil {
		return nil, err
	}

	nodeThroughput := make(map[string]float64)

	for _, id := range orderedIDs {
		node := perfNodes[id]
		incomingEdges := getIncomingEdgesFromList(node.ID, edges)

		if node.Type == NodeTypeSecurityGroup {
			if len(incomingEdges) == 0 {
				nodeThroughput[node.ID] = math.Inf(1)
			} else {
				nodeThroughput[node.ID] = sumIncomingFlow(incomingEdges, nodeThroughput)
			}
			continue
		}

		if len(incomingEdges) == 0 {
			nodeThroughput[node.ID] = node.MaxThroughput
			continue
		}

		incomingTotal := sumIncomingFlow(incomingEdges, nodeThroughput)
		nodeThroughput[node.ID] = math.Min(incomingTotal, node.MaxThroughput)
	}

	return findBottlenecksFromIntentGraph(perfNodes, edges, nodeThroughput), nil
}

// For testing purposes, print the bottlenecks in a readable format
func PrintBottlenecks(bottlenecks []Bottleneck) {
	if len(bottlenecks) == 0 {
		fmt.Println("No bottlenecks found.")
		return
	}

	fmt.Println("Performance Bottlenecks (sorted by drop):")
	for i, b := range bottlenecks {
		fmt.Printf("%d. %s Bottleneck at %s:\n", i+1, b.Type, b.ID)
		fmt.Printf("   Incoming Flow: %.2f\n", b.IncomingFlow)
		fmt.Printf("   Capacity: %.2f\n", b.Capacity)
		fmt.Printf("   Drop: %.2f\n", b.Drop)
		fmt.Println()
	}
}
