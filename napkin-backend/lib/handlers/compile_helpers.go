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

		attrs := map[string]string{}
		for k, v := range node.Attributes {
			if k == "" || isMetaAttributeKey(k) {
				continue
			}
			attrs[k] = v
		}

		ln := localNames[string(node.ID)]

		exprAttrs := exprDefaultsForKind(node.Kind, attrs)
		applyLocalNameDefaults(tfType, ln, attrs)

		var exprOut map[string]string
		if len(exprAttrs) > 0 {
			exprOut = exprAttrs
		}

		irNodes = append(irNodes, compiler.GraphNode{
			ID:             string(node.ID),
			LocalName:      ln,
			Kind:           string(node.Kind),
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

	return compiler.IR{Region: ig.Region, Nodes: irNodes, Edges: directed}, nil
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

// exprDefaultsForKind returns the HCL-expression defaults registered for a kind,
// filtered to keys not already present in attrs (a user-supplied plain attr wins).
func exprDefaultsForKind(kind graph.NodeKind, attrs map[string]string) map[string]string {
	def, ok := graph.Registry[kind]
	if !ok || len(def.ExprDefaults) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(def.ExprDefaults))
	for k, v := range def.ExprDefaults {
		if _, set := attrs[k]; set {
			continue
		}
		out[k] = v
	}
	return out
}

// applyLocalNameDefaults handles the small set of defaults that depend on the
// assigned Terraform local name (and so cannot live in the static Registry).
func applyLocalNameDefaults(tfType string, localName string, attrs map[string]string) {
	switch tfType {
	case "aws_lb":
		if attrs["name_prefix"] == "" && attrs["name"] == "" {
			prefix := localName
			if len(prefix) > 6 {
				prefix = prefix[:6]
			}
			if prefix == "" {
				prefix = "nk"
			}
			attrs["name_prefix"] = prefix
		}
	case "aws_s3_bucket":
		if attrs["bucket_prefix"] == "" {
			attrs["bucket_prefix"] = localName + "-"
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
