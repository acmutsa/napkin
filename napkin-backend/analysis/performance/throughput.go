package performance

import (
	"errors"
	"math"
	"napkin-backend/graph"
)

type NodeType string

const (
	NodeTypeSecurityGroup NodeType = "SECURITY_GROUP"
	NodeTypeLoadBalancer  NodeType = "LOAD_BALANCER"
	NodeTypeEC2           NodeType = "EC2"
	NodeTypeDatabase      NodeType = "DATABASE"
	NodeTypeStorageBucket NodeType = "STORAGE_BUCKET"
)

type Node struct {
	ID                 string
	Type               NodeType
	MaxThroughput      float64
	MaxReadThroughput  *float64
	MaxWriteThroughput *float64
	ReplicaCount       *int
	IsRequired         bool
	IsPassthrough      bool
}

type Edge struct {
	SourceNode     string
	TargetNode     string
	BandwidthLimit *float64
}

func getIntentGraphEdges(intentGraph *graph.DirectedGraph) []Edge {
	edges := make([]Edge, 0)
	for from, targets := range intentGraph.GetEdges() {
		for to := range targets {
			edges = append(edges, Edge{
				SourceNode: string(from),
				TargetNode: string(to),
			})
		}
	}
	return edges
}

func getIncomingEdgesFromList(nodeID string, edges []Edge) []Edge {
	var incoming []Edge
	for _, edge := range edges {
		if edge.TargetNode == nodeID {
			incoming = append(incoming, edge)
		}
	}
	return incoming
}

func getOutgoingEdgesFromList(nodeID string, edges []Edge) []Edge {
	var outgoing []Edge
	for _, edge := range edges {
		if edge.SourceNode == nodeID {
			outgoing = append(outgoing, edge)
		}
	}
	return outgoing
}

func topologicalSortIntentGraph(nodes map[string]Node, edges []Edge) ([]string, error) {
	inDegree := make(map[string]int)
	for id := range nodes {
		inDegree[id] = 0
	}
	for _, edge := range edges {
		inDegree[edge.TargetNode]++
	}

	queue := []string{}
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var ordered []string
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		ordered = append(ordered, curr)

		for _, edge := range edges {
			if edge.SourceNode == curr {
				inDegree[edge.TargetNode]--
				if inDegree[edge.TargetNode] == 0 {
					queue = append(queue, edge.TargetNode)
				}
			}
		}
	}

	if len(ordered) != len(nodes) {
		return nil, errors.New("intent graph contains a cycle or disconnected nodes")
	}

	return ordered, nil
}

func parsePerformanceNode(graphNode graph.Node) (Node, error) {
	perfNode := Node{
		ID: string(graphNode.ID),
	}

	if typeStr, ok := graphNode.Data["type"].(string); ok {
		switch typeStr {
		case "SECURITY_GROUP":
			perfNode.Type = NodeTypeSecurityGroup
		case "LOAD_BALANCER":
			perfNode.Type = NodeTypeLoadBalancer
		case "EC2":
			perfNode.Type = NodeTypeEC2
		case "DATABASE":
			perfNode.Type = NodeTypeDatabase
		case "STORAGE_BUCKET":
			perfNode.Type = NodeTypeStorageBucket
		default:
			perfNode.Type = NodeTypeEC2
		}
	} else {
		perfNode.Type = NodeTypeEC2
	}

	if maxThroughput, ok := graphNode.Data["maxThroughput"].(float64); ok {
		perfNode.MaxThroughput = maxThroughput
	} else if maxThroughputInt, ok := graphNode.Data["maxThroughput"].(int); ok {
		perfNode.MaxThroughput = float64(maxThroughputInt)
	}

	if maxRead, ok := graphNode.Data["maxReadThroughput"].(float64); ok {
		perfNode.MaxReadThroughput = &maxRead
	} else if maxReadInt, ok := graphNode.Data["maxReadThroughput"].(int); ok {
		maxReadFloat := float64(maxReadInt)
		perfNode.MaxReadThroughput = &maxReadFloat
	}

	if maxWrite, ok := graphNode.Data["maxWriteThroughput"].(float64); ok {
		perfNode.MaxWriteThroughput = &maxWrite
	} else if maxWriteInt, ok := graphNode.Data["maxWriteThroughput"].(int); ok {
		maxWriteFloat := float64(maxWriteInt)
		perfNode.MaxWriteThroughput = &maxWriteFloat
	}

	if replicaCount, ok := graphNode.Data["replicaCount"].(int); ok {
		perfNode.ReplicaCount = &replicaCount
	} else if replicaCountFloat, ok := graphNode.Data["replicaCount"].(float64); ok {
		replicaInt := int(replicaCountFloat)
		perfNode.ReplicaCount = &replicaInt
	}

	if isRequired, ok := graphNode.Data["isRequired"].(bool); ok {
		perfNode.IsRequired = isRequired
	}

	if isPassthrough, ok := graphNode.Data["isPassthrough"].(bool); ok {
		perfNode.IsPassthrough = isPassthrough
	}

	return perfNode, nil
}

func buildIntentGraphPerformanceNodes(intentGraph *graph.DirectedGraph) (map[string]Node, error) {
	perfNodes := make(map[string]Node)
	for id, graphNode := range intentGraph.GetNodes() {
		perfNode, err := parsePerformanceNode(graphNode)
		if err != nil {
			return nil, err
		}
		perfNodes[string(id)] = perfNode
	}
	return perfNodes, nil
}

func sumIncomingFlow(incomingEdges []Edge, nodeThroughput map[string]float64) float64 {
	total := 0.0
	for _, edge := range incomingEdges {
		upstream := nodeThroughput[edge.SourceNode]
		if edge.BandwidthLimit != nil {
			upstream = math.Min(upstream, *edge.BandwidthLimit)
		}
		total += upstream
	}
	return total
}

func resolveReplicas(node Node, mode string) int {
	if mode == "dev" {
		return 1
	}
	if node.ReplicaCount != nil {
		return *node.ReplicaCount
	}
	return 1
}

func AnalyzeThroughput(intentGraph *graph.DirectedGraph) (float64, error) {
	perfNodes, err := buildIntentGraphPerformanceNodes(intentGraph)
	if err != nil {
		return 0, err
	}

	edges := getIntentGraphEdges(intentGraph)

	orderedIDs, err := topologicalSortIntentGraph(perfNodes, edges)
	if err != nil {
		return 0, err
	}

	nodeThroughput := make(map[string]float64)

	for _, id := range orderedIDs {
		node := perfNodes[id]
		incomingEdges := getIncomingEdgesFromList(id, edges)

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

		var effectiveCapacity float64

		switch node.Type {
		case NodeTypeEC2:
			effectiveCapacity = node.MaxThroughput * float64(resolveReplicas(node, "prod"))

		case NodeTypeDatabase:
			effectiveCapacity = node.MaxThroughput * float64(resolveReplicas(node, "prod"))
			if node.MaxReadThroughput != nil || node.MaxWriteThroughput != nil {
				readCap := math.Inf(1)
				writeCap := math.Inf(1)
				if node.MaxReadThroughput != nil {
					readCap = *node.MaxReadThroughput
				}
				if node.MaxWriteThroughput != nil {
					writeCap = *node.MaxWriteThroughput
				}
				effectiveCapacity = math.Min(effectiveCapacity, math.Min(readCap, writeCap))
			}

		case NodeTypeStorageBucket:
			effectiveCapacity = node.MaxThroughput
			if node.MaxReadThroughput != nil || node.MaxWriteThroughput != nil {
				readCap := math.Inf(1)
				writeCap := math.Inf(1)
				if node.MaxReadThroughput != nil {
					readCap = *node.MaxReadThroughput
				}
				if node.MaxWriteThroughput != nil {
					writeCap = *node.MaxWriteThroughput
				}
				effectiveCapacity = math.Min(effectiveCapacity, math.Min(readCap, writeCap))
			}

		default:
			effectiveCapacity = node.MaxThroughput
		}

		nodeThroughput[node.ID] = math.Min(incomingTotal, effectiveCapacity)
	}

	var sinks []Node
	for id, node := range perfNodes {
		if len(getOutgoingEdgesFromList(id, edges)) == 0 {
			sinks = append(sinks, node)
		}
	}

	var requiredSinks []Node
	for _, s := range sinks {
		if s.IsRequired {
			requiredSinks = append(requiredSinks, s)
		}
	}
	if len(requiredSinks) == 0 {
		requiredSinks = sinks
	}

	result := math.Inf(1)
	for _, s := range requiredSinks {
		result = math.Min(result, nodeThroughput[s.ID])
	}
	return result, nil
}
