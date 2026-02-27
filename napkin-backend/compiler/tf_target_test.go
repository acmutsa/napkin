package compiler

import "testing"

func TestTerraformTargetCompile(t *testing.T) {
	ir := IR{
		Nodes: []GraphNode{
			{
				ID:    "web",
				Class: ClassResource,
				Type:  "aws_instance",
				Attributes: map[string]string{
					"ami":           "ami-123",
					"instance_type": "t2.micro",
				},
			},
		},
	}

	target := &TerraformTarget{}

	tfFile, err := target.Compile(ir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tfFile.Block) != 1 {
		t.Fatalf("expected 1 block, got %d", len(tfFile.Block))
	}

	block := tfFile.Block[0]

	if block.Class != "resource" {
		t.Errorf("expected class resource, got %s", block.Class)
	}

	if len(block.Labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(block.Labels))
	}

	if block.Labels[0] != "aws_instance" {
		t.Errorf("expected type aws_instance, got %s", block.Labels[0])
	}

	if block.Labels[1] != "web" {
		t.Errorf("expected name web, got %s", block.Labels[1])
	}

	if block.Attributes["ami"] != "ami-123" {
		t.Errorf("expected ami attribute to be ami-123")
	}
}
