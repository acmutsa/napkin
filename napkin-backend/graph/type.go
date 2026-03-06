package graph

type NodeID string

type Node struct {
	ID   NodeID         `json:"id"`
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

type Edge struct {
	From NodeID `json:"source"`
	To   NodeID `json:"target"`
}

type GraphJSON struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Graph interface {
	FromJSON(data []byte) error

	AddNode(node Node) error
	AddEdge(edge Edge) error

	HasNode(id NodeID) bool
	HasEdge(from, to NodeID) bool

	RemoveNode(id NodeID) error
	RemoveEdge(from, to NodeID) error
}
