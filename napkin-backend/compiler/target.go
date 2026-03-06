package compiler

type NodeClass string

const (
	ClassResource NodeClass = "resource"
	ClassProvider NodeClass = "provider"
	ClassModule   NodeClass = "module"
	ClassData     NodeClass = "data"
)

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
type TFBlock struct {
	Class      string
	Labels     []string
	Attributes map[string]string
	Blocks     []TFBlock
}
type TFFile struct {
	Block []TFBlock
}

type GraphNode struct {
	ID         string
	Class      NodeClass
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
