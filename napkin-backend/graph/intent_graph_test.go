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

	hasInNetwork := false
	for _, p := range got.Inputs {
		if p.ID == "in-network" && p.Type == PortNetwork {
			hasInNetwork = true
		}
	}
	if !hasInNetwork {
		t.Fatalf("compute should declare in-network input, got %#v", got.Inputs)
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
