package graph

// NodeKind is the canonical, semantic identifier for a node type on the canvas.
// It is independent of the Terraform resource string; analyzers should branch on
// Kind rather than spec.type.
type NodeKind string

const (
	KindCompute       NodeKind = "compute"
	KindDatabase      NodeKind = "database"
	KindLoadBalancer  NodeKind = "loadBalancer"
	KindSecurityGroup NodeKind = "securityGroup"
	KindStorageBucket NodeKind = "storageBucket"
	KindVPC           NodeKind = "vpc"
	KindSubnet        NodeKind = "subnet"
	KindIAMRole       NodeKind = "iamRole"
	KindLambda        NodeKind = "lambda"
	KindSQSQueue      NodeKind = "sqsQueue"
)

// PortType classifies what flows across an edge connecting two ports.
// Analyzers and the compiler use this to enforce that only same-type ports may
// be wired together.
type PortType string

const (
	PortNetwork PortType = "network"
	PortData    PortType = "data"
	PortEnv     PortType = "env"
	PortIAM     PortType = "iam"
)

// PortDef is one declared port (input or output) on a node kind.
//
// Port IDs are semantic (e.g. "subnet", "instanceRole", "connection") rather
// than direction-prefixed. Each ID is unique within a kind+direction so React
// Flow handle ids round-trip cleanly into edge.source.port / edge.target.port.
type PortDef struct {
	ID    string   `json:"id"`
	Type  PortType `json:"type"`
	Label string   `json:"label"`
}

// KindDef is the per-kind blueprint that drives both the IR (defaults, ports)
// and the Terraform compile target (TerraformType, ExprDefaults).
//
// Defaults are plain string attributes that are placed on Node.Attributes during
// Normalize so analyzers can inspect them. ExprDefaults are HCL-expression strings
// (e.g. "var.db_master_password" or "true") which the compile path stitches onto
// the IR's ExprAttributes; they intentionally do not become user-editable
// attributes.
type KindDef struct {
	TerraformType string
	Defaults      map[string]string
	ExprDefaults  map[string]string
	Inputs        []PortDef
	Outputs       []PortDef
}

// Registry is the single source of truth for node kinds on the backend.
// Keep in sync with napkin-app/src/lib/kinds.ts.
var Registry = map[NodeKind]KindDef{
	KindVPC: {
		TerraformType: "aws_vpc",
		Defaults: map[string]string{
			"cidr_block": "10.0.0.0/16",
		},
		Outputs: []PortDef{
			{ID: "network", Type: PortNetwork, Label: "Network"},
		},
	},
	KindSubnet: {
		TerraformType: "aws_subnet",
		Defaults: map[string]string{
			"cidr_block": "10.0.1.0/24",
		},
		Inputs: []PortDef{
			{ID: "vpc", Type: PortNetwork, Label: "Parent VPC"},
		},
		Outputs: []PortDef{
			{ID: "placement", Type: PortNetwork, Label: "Placement"},
		},
	},
	KindSecurityGroup: {
		TerraformType: "aws_security_group",
		Defaults: map[string]string{
			"cidr_blocks": `["0.0.0.0/0"]`,
		},
		Inputs: []PortDef{
			{ID: "vpc", Type: PortNetwork, Label: "VPC"},
		},
		Outputs: []PortDef{
			{ID: "attachment", Type: PortNetwork, Label: "Attach to"},
		},
	},
	KindIAMRole: {
		TerraformType: "aws_iam_role",
		// Minimal trust policy so generated HCL passes validation; override via canvas attributes if needed.
		ExprDefaults: map[string]string{
			"assume_role_policy": `jsonencode({Version = "2012-10-17", Statement = [{Effect = "Allow", Principal = {Service = "ec2.amazonaws.com"}, Action = "sts:AssumeRole"}]})`,
		},
		Outputs: []PortDef{
			{ID: "role", Type: PortIAM, Label: "Role"},
		},
	},
	KindCompute: {
		TerraformType: "aws_instance",
		Defaults: map[string]string{
			"instance_type": "t2.micro",
		},
		Inputs: []PortDef{
			{ID: "subnet", Type: PortNetwork, Label: "Subnet"},
			{ID: "securityGroup", Type: PortNetwork, Label: "Security Group"},
			{ID: "inboundTraffic", Type: PortNetwork, Label: "Inbound Traffic"},
			{ID: "instanceRole", Type: PortIAM, Label: "Instance Role"},
		},
		Outputs: []PortDef{
			{ID: "outboundTraffic", Type: PortNetwork, Label: "Outbound Traffic"},
			{ID: "dataSource", Type: PortData, Label: "Data Source"},
		},
	},
	KindDatabase: {
		TerraformType: "aws_db_instance",
		Defaults: map[string]string{
			"engine":         "mysql",
			"instance_class": "db.t3.micro",
			"username":       "admin",
		},
		ExprDefaults: map[string]string{
			"password":            "var.db_master_password",
			"allocated_storage":   "20",
			"skip_final_snapshot": "true",
		},
		Inputs: []PortDef{
			{ID: "subnet", Type: PortNetwork, Label: "Subnet"},
			{ID: "inboundTraffic", Type: PortNetwork, Label: "Inbound Traffic"},
			{ID: "connection", Type: PortData, Label: "DB Connection"},
		},
		Outputs: []PortDef{},
	},
	KindLoadBalancer: {
		TerraformType: "aws_lb",
		Defaults: map[string]string{
			"load_balancer_type": "application",
		},
		Inputs: []PortDef{
			{ID: "subnet", Type: PortNetwork, Label: "Subnet"},
			{ID: "securityGroup", Type: PortNetwork, Label: "Security Group"},
			{ID: "inboundTraffic", Type: PortNetwork, Label: "Public Traffic"},
		},
		Outputs: []PortDef{
			{ID: "forward", Type: PortNetwork, Label: "Forward to Targets"},
		},
	},
	KindLambda: {
		TerraformType: "aws_lambda_function",
		Defaults: map[string]string{
			"runtime":     "nodejs20.x",
			"memory_size": "128",
		},
		Inputs: []PortDef{
			{ID: "executionRole", Type: PortIAM, Label: "Execution Role"},
			{ID: "eventSource", Type: PortData, Label: "Event Source"},
		},
		Outputs: []PortDef{
			{ID: "output", Type: PortData, Label: "Output"},
		},
	},
	KindSQSQueue: {
		TerraformType: "aws_sqs_queue",
		Inputs: []PortDef{
			{ID: "producer", Type: PortData, Label: "Producer"},
		},
		Outputs: []PortDef{
			{ID: "consumer", Type: PortData, Label: "Consumer"},
		},
	},
	KindStorageBucket: {
		TerraformType: "aws_s3_bucket",
		Inputs: []PortDef{
			{ID: "producer", Type: PortData, Label: "Objects In"},
			{ID: "accessPolicy", Type: PortIAM, Label: "Access Policy"},
		},
		Outputs: []PortDef{
			{ID: "consumer", Type: PortData, Label: "Objects Out"},
		},
	},
}

// KindForTerraformType reverse-looks up a NodeKind by its registered Terraform type.
// Used as a best-effort fallback when payloads omit the explicit "kind" field.
func KindForTerraformType(tfType string) (NodeKind, bool) {
	for kind, def := range Registry {
		if def.TerraformType == tfType {
			return kind, true
		}
	}
	return "", false
}

// FindOutputPort returns the declared output port for a kind, or false.
func FindOutputPort(kind NodeKind, portID string) (PortDef, bool) {
	def, ok := Registry[kind]
	if !ok {
		return PortDef{}, false
	}
	for _, p := range def.Outputs {
		if p.ID == portID {
			return p, true
		}
	}
	return PortDef{}, false
}

// FindInputPort returns the declared input port for a kind, or false.
func FindInputPort(kind NodeKind, portID string) (PortDef, bool) {
	def, ok := Registry[kind]
	if !ok {
		return PortDef{}, false
	}
	for _, p := range def.Inputs {
		if p.ID == portID {
			return p, true
		}
	}
	return PortDef{}, false
}
