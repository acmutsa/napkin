package graph

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestAddAndHasNode(t *testing.T) {
	g := NewDirectedGraph()

	err := g.AddNode(Node{ID: "api"})
	if err != nil {
		t.Fatal(err)
	}

	if !g.HasNode("api") {
		t.Fatal("expected node api to exist")
	}
	fmt.Println(g)
}

func TestAddAndHasEdge(t *testing.T) {
	g := NewDirectedGraph()

	g.AddNode(Node{ID: "a"})
	g.AddNode(Node{ID: "b"})

	err := g.AddEdge(Edge{From: "a", To: "b"})
	if err != nil {
		t.Fatal(err)
	}

	if !g.HasEdge("a", "b") {
		t.Fatal("expected edge a -> b")
	}
}

func TestRemoveNodeRemovesEdges(t *testing.T) {
	g := NewDirectedGraph()

	g.AddNode(Node{ID: "a"})
	g.AddNode(Node{ID: "b"})
	g.AddEdge(Edge{From: "a", To: "b"})

	err := g.RemoveNode("b")
	if err != nil {
		t.Fatal(err)
	}

	if g.HasEdge("a", "b") {
		t.Fatal("edge should have been removed")
	}
}

func TestFromJSON(t *testing.T) {
	jsonData := []byte(`
	{
	  "nodes": [
	    { "id": "api", "type": "service" },
	    { "id": "db", "type": "postgres" }
	  ],
	  "edges": [
	    { "source": "api", "target": "db" }
	  ]
	}`)

	g := NewDirectedGraph()
	err := g.FromJSON(jsonData)
	if err != nil {
		t.Fatal(err)
	}

	if !g.HasEdge("api", "db") {
		t.Fatal("expected api -> db edge")
	}
	fmt.Println(g)
}

func TestIntentGraphWithSpring26Schema(t *testing.T) {
	rawJSON := `{
		"nodes": [
			{
				"id": "srv-01",
				"type": "Server",
				"data": {
					"instanceType": "t3.medium",
					"region": "us-east-1",
					"name": "Production-API",
					"ports": {
						"inputs": ["in-env", "in-network"],
						"outputs": ["out-network", "out-data"]
					}
				}
			},
			{
				"id": "db-01",
				"type": "Database",
				"data": {
					"engine": "postgres",
					"storageSize": 100,
					"region": "us-east-1",
					"ports": {
						"inputs": ["in-network"],
						"outputs": ["out-conn"]
					}
				}
			}
		],
		"edges": [
			{
				"source": "srv-01",
				"target": "db-01"
			}
		]
	}`

	ig := NewIntentGraph()

	err := ig.FromJSON([]byte(rawJSON))
	if err != nil {
		t.Fatalf("Failed to parse Spring '26 instance: %v", err)
	}

	prettyJSON, _ := json.MarshalIndent(ig, "", " ")
	fmt.Printf("IntentGraph Structure:\n%s\n", string(prettyJSON))

	srvNode, exists := ig.Nodes["srv-01"]
	if !exists {
		t.Fatal("Server node srv-01 was not found in IntentGraph")
	}

	if srvNode.Data["instanceType"] != "t3.medium" {
		t.Errorf("Expected instanceType t3.medium, got %v", srvNode.Data["instanceType"])
	}

	if !ig.HasEdge("srv-01", "db-01") {
		t.Error("Edge between Server and Database not found")
	}

	dg, err := ig.ToDirectedGraph()
	if err != nil {
		t.Fatalf("Failed to translate to DirectedGraph: %v", err)
	}

	if !dg.HasNode("db-01") {
		t.Error("DirectedGraph failed to import the Database node")
	}
}