package graph

type NodeID string

type Node struct {
	ID         NodeID            `json:"id"`
	Spec       any               `json:"spec"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type Edge struct {
	Source struct {
		Node string `json:"node"`
		ID   NodeID `json:"id"`
		Port string `json:"port"`
	} `json:"source"`
	Target struct {
		Node string `json:"node"`
		ID   NodeID `json:"id"`
		Port string `json:"port"`
	} `json:"target"`
	Type string `json:"type"`
}

type InnerGraph struct {
    Nodes map[string][]Node `json:"nodes"`
    Edges []Edge            `json:"edges"`
}

type GraphJSON struct {
    Type  string     `json:"type"`
    Graph InnerGraph `json:"graph"`
}

type NodeSpec struct {
	Label       string `json:"label"`
	Color       string `json:"color"`
	BorderColor string `json:"borderColor"`
}
