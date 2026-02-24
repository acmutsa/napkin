package compiler

type TFResource struct {
	Type       string
	Name       string
	Attributes map[string]string
}

type TFOutput struct {
	Name  string
	Value string
}

type TFModule struct {
	Name      string
	Resources []TFResource
}

type TFFile struct {
	Resources []TFResource
	Modules   []TFModule
	Outputs   []TFOutput
}

type GraphNode struct {
	ID         string
	Type       string
	Attributes map[string]string
	Edges      []string // list of node IDs this node depends on
}

type IR struct {
	Nodes []GraphNode
}

type Target interface {
	Compile(ir IR) (*TFFile, error)
}
