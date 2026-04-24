package handlers

import (
	"errors"
	"fmt"
	"napkin-backend/compiler"
	"napkin-backend/graph"
)

func IntentGraphToIR(ig *graph.IntentGraph) (compiler.IR, error) {
	ir := compiler.IR{
		Nodes: []compiler.GraphNode{},
	}

	nodes := ig.GetNodes()

	for _, node := range nodes {
		classVal, exists := node.Data["class"]
		if !exists {
			return ir, errors.New("missing class in node")
		}

		classStr, ok := classVal.(string)
		if !ok {
			return ir, errors.New("class must be a string")
		}

		class := compiler.NodeClass(classStr)

		attrs := map[string]string{}
		for attrName, dataVal := range node.Data {
			if attrName == "class" {
				continue
			}
			attrs[attrName] = fmt.Sprint(dataVal)
		}

		edges := ig.GetNeighborIDs(node.ID)
		edgeStrings := []string{}
		for _, edge := range edges {
			edgeStrings = append(edgeStrings, string(edge))
		}

		gNode := compiler.GraphNode{
			ID:         string(node.ID),
			Class:      class,
			Type:       node.Type,
			Attributes: attrs,
			Edges:      edgeStrings,
		}

		ir.Nodes = append(ir.Nodes, gNode)
	}

	return ir, nil
}
