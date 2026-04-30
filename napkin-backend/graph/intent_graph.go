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
    var payload InnerGraph // Change this from GraphJSON
    if err := json.Unmarshal(data, &payload); err != nil {
        return err
    }

    for _, nodeSlice := range payload.Nodes {
        for _, n := range nodeSlice {
            ig.Nodes[n.ID] = n
        }
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
        if e.Source.ID == from && e.Target.ID == to {
            return true
        }
    }
    return false
}

func (ig *IntentGraph) RemoveEdge(from, to NodeID) error {
    for i, e := range ig.Edges {
        if e.Source.ID == from && e.Target.ID == to {
            ig.Edges = append(ig.Edges[:i], ig.Edges[i+1:]...)
            return nil
        }
    }
    return errors.New("edge not found")
}

func (ig *IntentGraph) RemoveNode(id NodeID) error {
    if !ig.HasNode(id) {
        return errors.New("node not found")
    }
    delete(ig.Nodes, id)

    newEdges := []Edge{}
    for _, e := range ig.Edges {
        if e.Source.ID != id && e.Target.ID != id {
            newEdges = append(newEdges, e)
        }
    }
    ig.Edges = newEdges
    return nil
}

func (ig *IntentGraph) GetNeighborIDs(id NodeID) []NodeID {
    result := []NodeID{}
    for _, edge := range ig.Edges {
        if edge.Source.ID == id {
            result = append(result, edge.Target.ID)
        }
    }
    return result
}

func (ig *IntentGraph) GetNodes() []Node {
	result := make([]Node, 0, len(ig.Nodes))
	for _, node := range ig.Nodes {
		result = append(result, node)
	}
	return result
}