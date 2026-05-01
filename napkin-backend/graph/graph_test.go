package graph

import (
    "testing"
)

func TestAddAndHasNode(t *testing.T) {
    g := NewDirectedGraph()

    // Fixed: ID is a NodeID type
    err := g.AddNode(Node{ID: "api"})
    if err != nil {
        t.Fatal(err)
    }

    if !g.HasNode("api") {
        t.Fatal("expected node api to exist")
    }
}

func TestAddAndHasEdge(t *testing.T) {
    g := NewDirectedGraph()

    g.AddNode(Node{ID: "a"})
    g.AddNode(Node{ID: "b"})

    // FIXED: Use Source.ID and Target.ID
    e := Edge{}
    e.Source.ID = "a"
    e.Target.ID = "b"

    err := g.AddEdge(e)
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
    
    e := Edge{}
    e.Source.ID = "a"
    e.Target.ID = "b"
    g.AddEdge(e)

    err := g.RemoveNode("b")
    if err != nil {
        t.Fatal(err)
    }

    // Check that the edge is gone from the adjacency list
    if g.HasEdge("a", "b") {
        t.Fatal("edge should have been removed")
    }
}

func TestFromJSON(t *testing.T) {
    // FIXED: JSON must match the GraphJSON/IntentGraph structure
    // (Categorized nodes and nested source/target objects)
    jsonData := []byte(`
    {
      "graph": {
        "nodes": {
          "service": [
            { "id": "api", "spec": {"type": "service"} },
            { "id": "db", "spec": {"type": "postgres"} }
          ]
        },
        "edges": [
          { "source": {"id": "api"}, "target": {"id": "db"} }
        ]
      }
    }`)

    g := NewDirectedGraph()
    err := g.FromJSON(jsonData)
    if err != nil {
        t.Fatal(err)
    }

    if !g.HasEdge("api", "db") {
        t.Fatal("expected api -> db edge")
    }
}

func TestIntentGraphWithSpring26Schema(t *testing.T) {
    // IntentGraph.FromJSON expects InnerGraph (nodes + edges at top level).
    rawJSON := `{
            "nodes": {
                "resource": [
                    {
                        "id": "srv-01",
                        "spec": {
                            "instanceType": "t3.medium",
                            "name": "Production-API"
                        }
                    },
                    {
                        "id": "db-01",
                        "spec": {
                            "engine": "postgres"
                        }
                    }
                ]
            },
            "edges": [
                {
                    "source": { "id": "srv-01" },
                    "target": { "id": "db-01" }
                }
            ]
    }`

    ig := NewIntentGraph()

    err := ig.FromJSON([]byte(rawJSON))
    if err != nil {
        t.Fatalf("Failed to parse Spring '26 instance: %v", err)
    }

    srvNode, exists := ig.Nodes["srv-01"]
    if !exists {
        t.Fatal("Server node srv-01 was not found in IntentGraph")
    }

    // Type assertion is needed because Spec is 'any'
    specMap, ok := srvNode.Spec.(map[string]any)
    if !ok {
        t.Fatal("Node spec is not a map")
    }

    if specMap["instanceType"] != "t3.medium" {
        t.Errorf("Expected instanceType t3.medium, got %v", specMap["instanceType"])
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