package graph

type NodeID string

type Node struct {
	ID         NodeID            `json:"id"`
	Kind       NodeKind          `json:"kind,omitempty"`
	Spec       any               `json:"spec"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Ports      PortTopology      `json:"ports,omitempty"`
}

type PortTopology struct {
	Inputs  []PortDef `json:"inputs,omitempty"`
	Outputs []PortDef `json:"outputs,omitempty"`
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
	Region string            `json:"region,omitempty"`
	Nodes  map[string][]Node `json:"nodes"`
	Edges  []Edge            `json:"edges"`
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
