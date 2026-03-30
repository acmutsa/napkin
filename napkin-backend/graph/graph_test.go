package graph

import (
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