package performance

import (
	"napkin-backend/graph"
	"testing"
)

func TestAnalyzeThroughputWithIntentGraph(t *testing.T) {
	// Create intent graph: LB -> EC2 -> DB
	intentGraph := graph.NewDirectedGraph()

	// Add nodes with performance data
	intentGraph.AddNode(graph.Node{
		ID:   "lb",
		Type: "LOAD_BALANCER",
		Data: map[string]any{
			"type":          "LOAD_BALANCER",
			"maxThroughput": 1000.0,
		},
	})
	intentGraph.AddNode(graph.Node{
		ID:   "ec2",
		Type: "EC2",
		Data: map[string]any{
			"type":          "EC2",
			"maxThroughput": 500.0,
		},
	})
	intentGraph.AddNode(graph.Node{
		ID:   "db",
		Type: "DATABASE",
		Data: map[string]any{
			"type":          "DATABASE",
			"maxThroughput": 200.0,
		},
	})

	// Add edges
	intentGraph.AddEdge(graph.Edge{From: "lb", To: "ec2"})
	intentGraph.AddEdge(graph.Edge{From: "ec2", To: "db"})

	// Analyze throughput
	throughput, err := AnalyzeThroughput(intentGraph)
	if err != nil {
		t.Fatalf("AnalyzeThroughput failed: %v", err)
	}

	expected := 200.0 // bottleneck is DB
	if throughput != expected {
		t.Errorf("expected: %f, result: %f", expected, throughput)
	} else {
		t.Logf("expected: %f, result: %f", expected, throughput)
	}
}

func TestFindBottlenecksWithIntentGraph(t *testing.T) {
	// Create intent graph: LB -> EC2 (with bottleneck)
	intentGraph := graph.NewDirectedGraph()

	// Add nodes
	intentGraph.AddNode(graph.Node{
		ID:   "lb",
		Type: "LOAD_BALANCER",
		Data: map[string]any{
			"type":          "LOAD_BALANCER",
			"maxThroughput": 1000.0,
		},
	})
	intentGraph.AddNode(graph.Node{
		ID:   "ec2",
		Type: "EC2",
		Data: map[string]any{
			"type":          "EC2",
			"maxThroughput": 500.0,
		},
	})

	// Add edge
	intentGraph.AddEdge(graph.Edge{From: "lb", To: "ec2"})

	// Find bottlenecks
	bottlenecks, err := FindBottlenecks(intentGraph)
	if err != nil {
		t.Fatalf("FindBottlenecks failed: %v", err)
	}

	// Output the bottlenecks
	PrintBottlenecks(bottlenecks)

	if len(bottlenecks) != 1 {
		t.Errorf("expected: 1 bottleneck, result: %d", len(bottlenecks))
	} else {
		t.Logf("expected: 1 bottleneck, result: %d", len(bottlenecks))
	}

	if bottlenecks[0].Type != "node" || bottlenecks[0].ID != "ec2" {
		t.Errorf("expected: node bottleneck at ec2, result: %+v", bottlenecks[0])
	} else {
		t.Logf("expected: node bottleneck at ec2, result: %+v", bottlenecks[0])
	}
}

func TestAnalyzeThroughputWithReplicasInIntentGraph(t *testing.T) {
	// Create intent graph: LB -> 2x EC2 instances
	intentGraph := graph.NewDirectedGraph()

	intentGraph.AddNode(graph.Node{
		ID:   "lb",
		Type: "LOAD_BALANCER",
		Data: map[string]any{
			"type":          "LOAD_BALANCER",
			"maxThroughput": 1000.0,
		},
	})
	intentGraph.AddNode(graph.Node{
		ID:   "ec2",
		Type: "EC2",
		Data: map[string]any{
			"type":          "EC2",
			"maxThroughput": 500.0,
			"replicaCount":  2,
		},
	})

	intentGraph.AddEdge(graph.Edge{From: "lb", To: "ec2"})

	throughput, err := AnalyzeThroughput(intentGraph)
	if err != nil {
		t.Fatalf("AnalyzeThroughput failed: %v", err)
	}

	expected := 1000.0 // 500 * 2 replicas = 1000
	if throughput != expected {
		t.Errorf("expected: %f, result: %f", expected, throughput)
	} else {
		t.Logf("expected: %f, result: %f", expected, throughput)
	}
}
