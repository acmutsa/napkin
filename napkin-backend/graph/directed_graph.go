package graph

import (
    "encoding/json"
    "errors"
)

type DirectedGraph struct {
    nodes map[NodeID]Node
    // Adjacency list: map[SourceID]map[TargetID]struct{}
    edges map[NodeID]map[NodeID]struct{}
}

func NewDirectedGraph() *DirectedGraph {
    return &DirectedGraph{
        nodes: make(map[NodeID]Node),
        edges: make(map[NodeID]map[NodeID]struct{}),
    }
}

// Updated to handle the nested JSON structure correctly
func (g *DirectedGraph) FromJSON(data []byte) error {
    var payload GraphJSON
    if err := json.Unmarshal(data, &payload); err != nil {
        return err
    }

    // Flatten categorized nodes from payload.Graph.Nodes
    for _, nodeSlice := range payload.Graph.Nodes {
        for _, n := range nodeSlice {
            _ = g.AddNode(n)
        }
    }

    // Use payload.Graph.Edges
    for _, e := range payload.Graph.Edges {
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

func (g *DirectedGraph) AddEdge(edge Edge) error {
    // UPDATED: Reference edge.Source.ID and edge.Target.ID
    fromID := edge.Source.ID
    toID := edge.Target.ID

    if !g.HasNode(fromID) || !g.HasNode(toID) {
        return errors.New("both nodes must exist in the graph before adding an edge")
    }

    if g.edges[fromID] == nil {
        g.edges[fromID] = make(map[NodeID]struct{})
    }

    g.edges[fromID][toID] = struct{}{}
    return nil
}

func (g *DirectedGraph) RemoveNode(id NodeID) error {
    if !g.HasNode(id) {
        return errors.New("node does not exist")
    }

    // Remove the node metadata
    delete(g.nodes, id)
    
    // Remove all outgoing edges from this node
    delete(g.edges, id)

    // Remove all incoming edges to this node from other nodes
    for sourceID := range g.edges {
        delete(g.edges[sourceID], id)
    }

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

func (g *DirectedGraph) RemoveEdge(from, to NodeID) error {
    if !g.HasEdge(from, to) {
        return errors.New("edge does not exist")
    }
    delete(g.edges[from], to)
    return nil
}