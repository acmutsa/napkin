package graph

import (
	"encoding/json"
	"errors"
)

type DirectedGraph struct {
	nodes map[NodeID]Node
	edges map[NodeID]map[NodeID]struct{}
}

func NewDirectedGraph() *DirectedGraph {
	return &DirectedGraph{
		nodes: make(map[NodeID]Node),
		edges: make(map[NodeID]map[NodeID]struct{}),
	}
}

func (g *DirectedGraph) FromJSON(data []byte) error {
	var payload GraphJSON
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	for _, n := range payload.Nodes {
		_ = g.AddNode(n)
	}

	for _, e := range payload.Edges {
		if err := g.AddEdge(e); err != nil {
			return err
		}
	}

	return nil
}

func (g *DirectedGraph) AddNode(node Node) error {
	if node.ID == "" {
		return errors.New("node id cannot be empty")
	}
	g.nodes[node.ID] = node
	return nil
}

func (g *DirectedGraph) RemoveNode(id NodeID) error {
	if !g.HasNode(id) {
		return errors.New("node does not exist")
	}

	delete(g.nodes, id)
	delete(g.edges, id)

	for _, neighbors := range g.edges {
		delete(neighbors, id)
	}

	return nil
}

func (g *DirectedGraph) AddEdge(edge Edge) error {
	if !g.HasNode(edge.From) || !g.HasNode(edge.To) {
		return errors.New("both nodes must exist")
	}

	if g.edges[edge.From] == nil {
		g.edges[edge.From] = make(map[NodeID]struct{})
	}

	g.edges[edge.From][edge.To] = struct{}{}
	return nil
}

func (g *DirectedGraph) RemoveEdge(from, to NodeID) error {
	if !g.HasEdge(from, to) {
		return errors.New("edge does not exist")
	}
	delete(g.edges[from], to)
	return nil
}

func (g *DirectedGraph) HasNode(id NodeID) bool {
	_, ok := g.nodes[id]
	return ok
}

func (g *DirectedGraph) HasEdge(from, to NodeID) bool {
	neighbors, ok := g.edges[from]
	if !ok {
		return false
	}
	_, exists := neighbors[to]
	return exists
}

func (g *DirectedGraph) GetNodes() map[NodeID]Node {
	return g.nodes
}

func (g *DirectedGraph) GetEdges() map[NodeID]map[NodeID]struct{} {
	return g.edges
}
