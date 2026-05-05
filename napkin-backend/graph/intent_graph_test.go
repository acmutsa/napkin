package graph

import (
	"testing"
)

func TestNormalize_FillsRegionDefault(t *testing.T) {
	ig := NewIntentGraph()
	if err := ig.Normalize(); err != nil {
		t.Fatal(err)
	}
	if ig.Region != defaultRegion {
		t.Fatalf("Region=%q want %q", ig.Region, defaultRegion)
	}
}

func TestNormalize_PreservesExplicitRegion(t *testing.T) {
	ig := NewIntentGraph()
	ig.Region = "eu-west-1"
	if err := ig.Normalize(); err != nil {
		t.Fatal(err)
	}
	if ig.Region != "eu-west-1" {
		t.Fatalf("Region=%q want eu-west-1", ig.Region)
	}
}

func TestNormalize_BackfillsComputeDefaults(t *testing.T) {
	ig := NewIntentGraph()
	ig.Nodes["ec2"] = Node{
		ID:   "ec2",
		Kind: KindCompute,
		Spec: map[string]any{"label": "EC2 Instance"},
	}

	if err := ig.Normalize(); err != nil {
		t.Fatal(err)
	}

	got := ig.Nodes["ec2"]
	if got.Attributes["instance_type"] != "t2.micro" {
		t.Fatalf("instance_type=%q want t2.micro", got.Attributes["instance_type"])
	}

	specMap, ok := got.Spec.(map[string]any)
	if !ok {
		t.Fatalf("spec not a map after Normalize")
	}
	if specMap["type"] != "aws_instance" {
		t.Fatalf("spec.type=%v want aws_instance", specMap["type"])
	}
	if specMap["class"] != "resource" {
		t.Fatalf("spec.class=%v want resource", specMap["class"])
	}
}

func TestNormalize_UserAttributesWinOverDefaults(t *testing.T) {
	ig := NewIntentGraph()
	ig.Nodes["ec2"] = Node{
		ID:         "ec2",
		Kind:       KindCompute,
		Spec:       map[string]any{"label": "EC2 Instance"},
		Attributes: map[string]string{"instance_type": "t3.large"},
	}

	if err := ig.Normalize(); err != nil {
		t.Fatal(err)
	}
	if ig.Nodes["ec2"].Attributes["instance_type"] != "t3.large" {
		t.Fatalf("user override clobbered: %#v", ig.Nodes["ec2"].Attributes)
	}
}

func TestNormalize_BackfillsSecurityGroupCidrBlocks(t *testing.T) {
	ig := NewIntentGraph()
	ig.Nodes["sg"] = Node{
		ID:   "sg",
		Kind: KindSecurityGroup,
		Spec: map[string]any{"label": "Security Group"},
	}

	if err := ig.Normalize(); err != nil {
		t.Fatal(err)
	}
	got := ig.Nodes["sg"].Attributes["cidr_blocks"]
	if got != `["0.0.0.0/0"]` {
		t.Fatalf("cidr_blocks=%q want default open CIDR", got)
	}
}

func TestNormalize_PopulatesPortsFromRegistry(t *testing.T) {
	ig := NewIntentGraph()
	ig.Nodes["ec2"] = Node{
		ID:   "ec2",
		Kind: KindCompute,
		Spec: map[string]any{"label": "EC2 Instance"},
	}

	if err := ig.Normalize(); err != nil {
		t.Fatal(err)
	}
	got := ig.Nodes["ec2"].Ports
	if len(got.Inputs) == 0 || len(got.Outputs) == 0 {
		t.Fatalf("expected ports populated, got %#v", got)
	}

	hasSubnet := false
	for _, p := range got.Inputs {
		if p.ID == "subnet" && p.Type == PortNetwork {
			hasSubnet = true
		}
	}
	if !hasSubnet {
		t.Fatalf("compute should declare subnet input, got %#v", got.Inputs)
	}
}

func TestValidateEdges_OK(t *testing.T) {
	ig := NewIntentGraph()
	ig.Nodes["vpc"] = Node{ID: "vpc", Kind: KindVPC, Spec: map[string]any{}}
	ig.Nodes["sn"] = Node{ID: "sn", Kind: KindSubnet, Spec: map[string]any{}}
	e := Edge{Type: "data-flow"}
	e.Source.ID, e.Source.Port = "vpc", "network"
	e.Target.ID, e.Target.Port = "sn", "vpc"
	ig.Edges = []Edge{e}
	if err := ig.Normalize(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateEdges_MissingSourcePort(t *testing.T) {
	ig := NewIntentGraph()
	ig.Nodes["vpc"] = Node{ID: "vpc", Kind: KindVPC, Spec: map[string]any{}}
	ig.Nodes["sn"] = Node{ID: "sn", Kind: KindSubnet, Spec: map[string]any{}}
	e := Edge{}
	e.Source.ID = "vpc"
	e.Target.ID, e.Target.Port = "sn", "vpc"
	ig.Edges = []Edge{e}
	err := ig.Normalize()
	if err == nil {
		t.Fatal("expected error for missing source port")
	}
}

func TestValidateEdges_MissingTargetPort(t *testing.T) {
	ig := NewIntentGraph()
	ig.Nodes["vpc"] = Node{ID: "vpc", Kind: KindVPC, Spec: map[string]any{}}
	ig.Nodes["sn"] = Node{ID: "sn", Kind: KindSubnet, Spec: map[string]any{}}
	e := Edge{}
	e.Source.ID, e.Source.Port = "vpc", "network"
	e.Target.ID = "sn"
	ig.Edges = []Edge{e}
	if err := ig.Normalize(); err == nil {
		t.Fatal("expected error for missing target port")
	}
}

func TestValidateEdges_UnknownPortName(t *testing.T) {
	ig := NewIntentGraph()
	ig.Nodes["vpc"] = Node{ID: "vpc", Kind: KindVPC, Spec: map[string]any{}}
	ig.Nodes["sn"] = Node{ID: "sn", Kind: KindSubnet, Spec: map[string]any{}}
	e := Edge{}
	e.Source.ID, e.Source.Port = "vpc", "network"
	e.Target.ID, e.Target.Port = "sn", "not-a-real-port"
	ig.Edges = []Edge{e}
	if err := ig.Normalize(); err == nil {
		t.Fatal("expected error for unknown target port")
	}
}

func TestValidateEdges_PortTypeMismatch(t *testing.T) {
	ig := NewIntentGraph()
	ig.Nodes["role"] = Node{ID: "role", Kind: KindIAMRole, Spec: map[string]any{}}
	ig.Nodes["sn"] = Node{ID: "sn", Kind: KindSubnet, Spec: map[string]any{}}
	e := Edge{}
	e.Source.ID, e.Source.Port = "role", "role"
	e.Target.ID, e.Target.Port = "sn", "vpc"
	ig.Edges = []Edge{e}
	if err := ig.Normalize(); err == nil {
		t.Fatal("expected error for iam -> network port type mismatch")
	}
}

func TestNormalize_PreservesExplicitPorts(t *testing.T) {
	custom := []PortDef{{ID: "custom-in", Type: PortData, Label: "Custom"}}
	ig := NewIntentGraph()
	ig.Nodes["ec2"] = Node{
		ID:   "ec2",
		Kind: KindCompute,
		Spec: map[string]any{"label": "EC2 Instance"},
		Ports: PortTopology{
			Inputs: custom,
		},
	}

	if err := ig.Normalize(); err != nil {
		t.Fatal(err)
	}
	got := ig.Nodes["ec2"].Ports.Inputs
	if len(got) != 1 || got[0].ID != "custom-in" {
		t.Fatalf("explicit ports overwritten: %#v", got)
	}
}

func TestNormalize_InfersKindFromTerraformType(t *testing.T) {
	ig := NewIntentGraph()
	ig.Nodes["ec2"] = Node{
		ID:   "ec2",
		Spec: map[string]any{"label": "EC2 Instance", "type": "aws_instance"},
	}

	if err := ig.Normalize(); err != nil {
		t.Fatal(err)
	}
	if ig.Nodes["ec2"].Kind != KindCompute {
		t.Fatalf("Kind=%q want %q", ig.Nodes["ec2"].Kind, KindCompute)
	}
	if ig.Nodes["ec2"].Attributes["instance_type"] != "t2.micro" {
		t.Fatalf("defaults missing after kind inference: %#v", ig.Nodes["ec2"].Attributes)
	}
}

func TestNormalize_UnknownKindReturnsError(t *testing.T) {
	ig := NewIntentGraph()
	ig.Nodes["x"] = Node{
		ID:   "x",
		Kind: NodeKind("not-a-real-kind"),
		Spec: map[string]any{},
	}
	if err := ig.Normalize(); err == nil {
		t.Fatal("expected error for unknown kind")
	}
}

func TestNormalize_UnresolvableKindReturnsError(t *testing.T) {
	ig := NewIntentGraph()
	ig.Nodes["x"] = Node{
		ID:   "x",
		Spec: map[string]any{"label": "Mystery"},
	}
	if err := ig.Normalize(); err == nil {
		t.Fatal("expected error when kind cannot be inferred")
	}
}
