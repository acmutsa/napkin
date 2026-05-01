package graph

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const defaultRegion = "us-east-1"

type IntentGraph struct {
	Region string
	Nodes  map[NodeID]Node
	Edges  []Edge
}

func NewIntentGraph() *IntentGraph {
	return &IntentGraph{
		Nodes: make(map[NodeID]Node),
		Edges: []Edge{},
	}
}

func (ig *IntentGraph) FromJSON(data []byte) error {
	if ig.Nodes == nil {
		ig.Nodes = make(map[NodeID]Node)
	}

	var payload InnerGraph // Change this from GraphJSON
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	ig.Region = payload.Region

	for _, nodeSlice := range payload.Nodes {
		for _, n := range nodeSlice {
			ig.Nodes[n.ID] = n
		}
	}

	ig.Edges = payload.Edges
	return nil
}

// Normalize backfills derived fields on the IntentGraph so downstream
// consumers (analyzers, compilers) can rely on a uniform shape:
//
//   - Region defaults to "us-east-1" when missing.
//   - Each Node.Kind is resolved (explicit, or inferred from spec.type) and
//     looked up against Registry. Unknown kinds return an error.
//   - Missing Node.Attributes keys listed in KindDef.Defaults are filled.
//   - Node.Ports is populated from KindDef when empty so analyzers can see
//     the declared port surface, not just the edges that happen to use it.
//   - spec.class and spec.type are filled from the registry when missing,
//     keeping IntentGraphToIR happy without changing its contract.
func (ig *IntentGraph) Normalize() error {
	if strings.TrimSpace(ig.Region) == "" {
		ig.Region = defaultRegion
	}

	for id, node := range ig.Nodes {
		specMap, _ := node.Spec.(map[string]any)
		if specMap == nil {
			specMap = map[string]any{}
		}

		kind := node.Kind
		if kind == "" {
			if k, ok := inferKindFromSpec(specMap); ok {
				kind = k
			}
		}
		if kind == "" {
			return fmt.Errorf("node %s: cannot determine kind (set node.kind or spec.type)", id)
		}

		def, ok := Registry[kind]
		if !ok {
			return fmt.Errorf("node %s: unknown kind %q", id, kind)
		}

		if _, exists := specMap["class"]; !exists {
			specMap["class"] = "resource"
		}
		if _, exists := specMap["type"]; !exists {
			specMap["type"] = def.TerraformType
		}

		if node.Attributes == nil {
			node.Attributes = map[string]string{}
		}
		for k, v := range def.Defaults {
			if _, set := node.Attributes[k]; !set {
				node.Attributes[k] = v
			}
		}

		if len(node.Ports.Inputs) == 0 && len(def.Inputs) > 0 {
			node.Ports.Inputs = append([]PortDef(nil), def.Inputs...)
		}
		if len(node.Ports.Outputs) == 0 && len(def.Outputs) > 0 {
			node.Ports.Outputs = append([]PortDef(nil), def.Outputs...)
		}

		node.Kind = kind
		node.Spec = specMap
		ig.Nodes[id] = node
	}

	return nil
}

func inferKindFromSpec(specMap map[string]any) (NodeKind, bool) {
	tfType, _ := specMap["type"].(string)
	tfType = strings.TrimSpace(tfType)
	if tfType == "" {
		return "", false
	}
	return KindForTerraformType(tfType)
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
