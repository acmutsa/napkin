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
	// Inheritance records implicit attribute bindings the compiler inferred
	// (e.g. an LB whose subnets were inherited from its EC2 targets, or a
	// subnet whose vpc_id was inferred from the single canvas VPC). Keyed by
	// node ID then by HCL field name, with a human-readable source string
	// (typically a comma-separated list of source LocalNames). Not rendered
	// in the HCL output; used by the API to surface "inherited from..." UI hints.
	Inheritance map[string]map[string]string
}

// DirectedEdge is a canvas edge (React Flow IDs). Ports match handle ids
// declared on the node kind (e.g. "subnet", "instanceRole", "connection"); the
// compile target dispatches wiring effects based on
// (FromKind+SourcePort, ToKind+TargetPort), so both ports are required.
type DirectedEdge struct {
	FromID, ToID           string
	SourcePort, TargetPort string
}

type GraphNode struct {
	ID             string
	LocalName      string // Terraform resource/data local name (slug)
	Kind           string // canonical kind id, mirrors graph.NodeKind
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
