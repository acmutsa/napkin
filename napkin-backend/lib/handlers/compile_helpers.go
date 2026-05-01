package handlers

import (
	"fmt"
	"napkin-backend/compiler"
	"napkin-backend/graph"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

func IntentGraphToIR(ig *graph.IntentGraph) (compiler.IR, error) {
	nodes := ig.GetNodes()
	slices.SortFunc(nodes, func(a, b graph.Node) int {
		return strings.Compare(string(a.ID), string(b.ID))
	})

	localNames := assignLocalNames(nodes)

	irNodes := make([]compiler.GraphNode, 0, len(nodes))
	for _, node := range nodes {
		specMap, ok := node.Spec.(map[string]any)
		if !ok {
			specMap = make(map[string]any)
		}

		classVal, exists := specMap["class"]
		if !exists {
			return compiler.IR{}, fmt.Errorf("node %s missing class in spec", node.ID)
		}
		classStr, ok := classVal.(string)
		if !ok {
			return compiler.IR{}, fmt.Errorf("node %s class must be a string", node.ID)
		}

		tfType := strings.TrimSpace(fmt.Sprint(specMap["type"]))
		if tfType == "" {
			return compiler.IR{}, fmt.Errorf("node %s missing type in spec", node.ID)
		}

		attrs := terraformAttrsFromWhitelist(specMap)

		for k, v := range node.Attributes {
			if k == "" || isMetaAttributeKey(k) {
				continue
			}
			attrs[k] = v
		}

		exprAttrs := map[string]string{}
		applyTerraformDefaults(tfType, attrs, exprAttrs)

		ln := localNames[string(node.ID)]

		var exprOut map[string]string
		if len(exprAttrs) > 0 {
			exprOut = exprAttrs
		}

		irNodes = append(irNodes, compiler.GraphNode{
			ID:             string(node.ID),
			LocalName:      ln,
			Class:          compiler.NodeClass(classStr),
			Type:           tfType,
			Attributes:     attrs,
			ExprAttributes: exprOut,
		})
	}

	directed := make([]compiler.DirectedEdge, 0, len(ig.Edges))
	for _, e := range ig.Edges {
		directed = append(directed, compiler.DirectedEdge{
			FromID:     string(e.Source.ID),
			ToID:       string(e.Target.ID),
			SourcePort: e.Source.Port,
			TargetPort: e.Target.Port,
		})
	}

	return compiler.IR{Nodes: irNodes, Edges: directed}, nil
}

// terraformAttrsFromWhitelist copies only explicit Terraform argument keys from spec.
// type, class, label are handled elsewhere and must never become resource attributes.
func terraformAttrsFromWhitelist(specMap map[string]any) map[string]string {
	allowed := map[string]struct{}{
		// extend when canvas sends TF keys through spec
	}
	out := map[string]string{}
	for k := range allowed {
		if v, ok := specMap[k]; ok {
			out[k] = fmt.Sprint(v)
		}
	}
	return out
}

// isMetaAttributeKey drops canvas/IR keys users might paste into the attribute editor.
func isMetaAttributeKey(k string) bool {
	switch strings.ToLower(strings.TrimSpace(k)) {
	case "label", "type", "class", "color", "bordercolor", "iconcolor":
		return true
	default:
		return false
	}
}

func applyTerraformDefaults(tfType string, attrs map[string]string, exprAttrs map[string]string) {
	switch tfType {
	case "aws_instance":
		if attrs["ami"] == "" {
			attrs["ami"] = "ami-123"
		}
		if attrs["instance_type"] == "" {
			attrs["instance_type"] = "t2.micro"
		}

	case "aws_db_instance":
		if attrs["engine"] == "" {
			attrs["engine"] = "mysql"
		}
		if attrs["instance_class"] == "" {
			attrs["instance_class"] = "db.t3.micro"
		}
		if attrs["username"] == "" {
			attrs["username"] = "admin"
		}
		if attrs["password"] == "" {
			attrs["password"] = "changeme_replace_in_prod"
		}
		if attrs["allocated_storage"] == "" && exprAttrs["allocated_storage"] == "" {
			exprAttrs["allocated_storage"] = "20"
		}
		if attrs["skip_final_snapshot"] == "" && exprAttrs["skip_final_snapshot"] == "" {
			exprAttrs["skip_final_snapshot"] = "true"
		}
	}
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

func assignLocalNames(nodes []graph.Node) map[string]string {
	result := make(map[string]string, len(nodes))
	used := make(map[string]bool)

	for _, node := range nodes {
		specMap, _ := node.Spec.(map[string]any)
		label := ""
		if l, ok := specMap["label"].(string); ok {
			label = l
		}
		base := slugifyLabel(label)
		if base == "" {
			base = "resource"
		}
		result[string(node.ID)] = uniqueSlug(base, used)
	}
	return result
}

func slugifyLabel(label string) string {
	s := strings.ToLower(strings.TrimSpace(label))
	s = nonSlugChars.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	if s == "" {
		return ""
	}
	r0 := rune(s[0])
	if unicode.IsDigit(r0) {
		s = "r_" + s
	}
	return s
}

func uniqueSlug(base string, used map[string]bool) string {
	name := base
	for suffix := 2; used[name]; suffix++ {
		name = fmt.Sprintf("%s_%d", base, suffix)
	}
	used[name] = true
	return name
}
