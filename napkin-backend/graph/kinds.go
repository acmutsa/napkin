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
// Analyzers use this to reason about traffic vs. data vs. configuration vs. identity.
type PortType string

const (
	PortNetwork PortType = "network"
	PortData    PortType = "data"
	PortEnv     PortType = "env"
	PortIAM     PortType = "iam"
)

// PortDef is one declared port (input or output) on a node kind.
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
	KindCompute: {
		TerraformType: "aws_instance",
		Defaults: map[string]string{
			"instance_type": "t2.micro",
		},
		Inputs: []PortDef{
			{ID: "in-network", Type: PortNetwork, Label: "Incoming Traffic"},
			{ID: "in-env", Type: PortEnv, Label: "Environment"},
			{ID: "in-iam", Type: PortIAM, Label: "IAM Role"},
		},
		Outputs: []PortDef{
			{ID: "out-network", Type: PortNetwork, Label: "Outbound Traffic"},
			{ID: "out-data", Type: PortData, Label: "Data Out"},
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
			{ID: "in-network", Type: PortNetwork, Label: "Network"},
			{ID: "in-env", Type: PortEnv, Label: "Configuration"},
		},
		Outputs: []PortDef{
			{ID: "out-data", Type: PortData, Label: "Connection"},
		},
	},
	KindLoadBalancer: {
		TerraformType: "aws_lb",
		Defaults: map[string]string{
			"load_balancer_type": "application",
		},
		Inputs: []PortDef{
			{ID: "in-network", Type: PortNetwork, Label: "Incoming Traffic"},
		},
		Outputs: []PortDef{
			{ID: "out-network", Type: PortNetwork, Label: "Forward Traffic"},
		},
	},
	KindSecurityGroup: {
		TerraformType: "aws_security_group",
		Defaults: map[string]string{
			"cidr_blocks": `["0.0.0.0/0"]`,
		},
		Inputs: []PortDef{
			{ID: "in-network", Type: PortNetwork, Label: "Inbound"},
		},
		Outputs: []PortDef{
			{ID: "out-network", Type: PortNetwork, Label: "Outbound"},
		},
	},
	KindStorageBucket: {
		TerraformType: "aws_s3_bucket",
		Inputs: []PortDef{
			{ID: "in-data", Type: PortData, Label: "Objects In"},
			{ID: "in-iam", Type: PortIAM, Label: "Access Policy"},
		},
		Outputs: []PortDef{
			{ID: "out-data", Type: PortData, Label: "Objects Out"},
		},
	},
	KindVPC: {
		TerraformType: "aws_vpc",
		Defaults: map[string]string{
			"cidr_block": "10.0.0.0/16",
		},
		Outputs: []PortDef{
			{ID: "out-network", Type: PortNetwork, Label: "Network"},
		},
	},
	KindSubnet: {
		TerraformType: "aws_subnet",
		Defaults: map[string]string{
			"cidr_block": "10.0.1.0/24",
		},
		Inputs: []PortDef{
			{ID: "in-network", Type: PortNetwork, Label: "Parent VPC"},
		},
		Outputs: []PortDef{
			{ID: "out-network", Type: PortNetwork, Label: "Network"},
		},
	},
	KindIAMRole: {
		TerraformType: "aws_iam_role",
		Outputs: []PortDef{
			{ID: "out-iam", Type: PortIAM, Label: "Role"},
		},
	},
	KindLambda: {
		TerraformType: "aws_lambda_function",
		Defaults: map[string]string{
			"runtime":     "nodejs20.x",
			"memory_size": "128",
		},
		Inputs: []PortDef{
			{ID: "in-iam", Type: PortIAM, Label: "Execution Role"},
			{ID: "in-env", Type: PortEnv, Label: "Environment"},
			{ID: "in-data", Type: PortData, Label: "Event Source"},
		},
		Outputs: []PortDef{
			{ID: "out-network", Type: PortNetwork, Label: "Outbound Calls"},
			{ID: "out-data", Type: PortData, Label: "Data Out"},
		},
	},
	KindSQSQueue: {
		TerraformType: "aws_sqs_queue",
		Inputs: []PortDef{
			{ID: "in-data", Type: PortData, Label: "Producer"},
			{ID: "in-iam", Type: PortIAM, Label: "Access Policy"},
		},
		Outputs: []PortDef{
			{ID: "out-data", Type: PortData, Label: "Consumer"},
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
