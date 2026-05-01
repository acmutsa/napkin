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
	// RawLine, when non-empty, is emitted as a single line (e.g. section banners). No braces.
	RawLine        string
	Class          string
	Labels         []string
	Attributes     map[string]string // string literals (quoted in HCL)
	ExprAttributes map[string]string // raw RHS expressions (unquoted)
	Blocks         []TFBlock
}

type TFFile struct {
	Block []TFBlock
}

// DirectedEdge is a canvas edge (React Flow IDs). Ports match handle ids (e.g. db-in, db-out).
type DirectedEdge struct {
	FromID, ToID             string
	SourcePort, TargetPort string
}

type GraphNode struct {
	ID             string
	LocalName      string // Terraform resource/data local name (slug)
	Class          NodeClass
	Type           string // Terraform type e.g. aws_instance
	Attributes     map[string]string
	ExprAttributes map[string]string // numeric/bool defaults, etc.
}

type IR struct {
	Region string // AWS region; default us-east-1 when empty
	Nodes  []GraphNode
	Edges  []DirectedEdge
}

type Target interface {
	Compile(ir IR) (*TFFile, error)
}
