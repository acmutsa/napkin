package graph

import (
	"encoding/json"
	"errors"
)

type IntentGraph struct {
	Nodes map[NodeID]Node
	Edges []Edge
}

func NewIntentGraph() *IntentGraph {
	return &IntentGraph{
		Nodes: make(map[NodeID]Node),
		Edges: []Edge{},
	}
}

func (ig *IntentGraph) FromJSON(data []byte) error {
	var payload GraphJSON
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	for _, n := range payload.Nodes {
		ig.Nodes[n.ID] = n
	}
	ig.Edges = payload.Edges
	return nil
}

func (ig *IntentGraph) ToDirectedGraph() (*DirectedGraph, error) {
	dg := NewDirectedGraph()

	for _, node := range ig.Nodes {
		if err := dg.AddNode(node); err != nil {
			return nil, err
		}
	}

	for _, edge := range ig.Edges {
		if err := dg.AddEdge(edge); err != nil {
			return nil, err
		}
	}

	return dg, nil
}

func (ig *IntentGraph) AddNode(node Node) error {
	ig.Nodes[node.ID] = node
	return nil
}

func (ig *IntentGraph) AddEdge(edge Edge) error {
	ig.Edges = append(ig.Edges, edge)
	return nil
}

func (ig *IntentGraph) HasNode(id NodeID) bool {
	_, ok := ig.Nodes[id]
	return ok
}

func (ig *IntentGraph) HasEdge(from, to NodeID) bool {
	for _, e := range ig.Edges {
		if e.From == from && e.To == to {
			return true
		}
	}
	return false
}

func (ig *IntentGraph) RemoveNode(id NodeID) error {
	delete(ig.Nodes, id)
	// Remove associated edges
	newEdges := []Edge{}
	for _, e := range ig.Edges {
		if e.From != id && e.To != id {
			newEdges = append(newEdges, e)
		}
	}
	ig.Edges = newEdges
	return nil
}

func (ig *IntentGraph) RemoveEdge(from, to NodeID) error {
	for i, e := range ig.Edges {
		if e.From == from && e.To == to {
			ig.Edges = append(ig.Edges[:i], ig.Edges[i+1:]...)
			return nil
		}
	}
	return errors.New("edge not found")
}