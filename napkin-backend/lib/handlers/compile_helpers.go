package handlers

import (
	"fmt"
	"napkin-backend/compiler"
	"napkin-backend/graph"
)

func IntentGraphToIR(ig *graph.IntentGraph) (compiler.IR, error) {
    ir := compiler.IR{
        Nodes: []compiler.GraphNode{},
    }

    for _, node := range ig.GetNodes() {
        specMap, ok := node.Spec.(map[string]any)
        if !ok {
            specMap = make(map[string]any)
        }

        classVal, exists := specMap["class"]
        if !exists {
            return ir, fmt.Errorf("node %s missing class in spec", node.ID)
        }
        classStr, ok := classVal.(string)
        if !ok {
            return ir, fmt.Errorf("node %s class must be a string", node.ID)
        }

        attrs := map[string]string{}
        for attrName, dataVal := range specMap {
            if attrName == "class" {
                continue
            }
            attrs[attrName] = fmt.Sprint(dataVal)
        }

        gNode := compiler.GraphNode{
            ID:         string(node.ID),
            Class:      compiler.NodeClass(classStr),
            Type:       fmt.Sprint(specMap["type"]), 
            Attributes: attrs,
            Edges:      toStringSlice(ig.GetNeighborIDs(node.ID)),
        }

        ir.Nodes = append(ir.Nodes, gNode)
    }

    return ir, nil
}

func toStringSlice(ids []graph.NodeID) []string {
    s := make([]string, len(ids))
    for i, id := range ids {
        s[i] = string(id)
    }
    return s
}