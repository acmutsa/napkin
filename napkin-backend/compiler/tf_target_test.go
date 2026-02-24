package compiler

import "testing"

func TestTerraformTargetCompile(t *testing.T) {
	ir := IR{
		Nodes: []GraphNode{
			{
				ID:   "web",
				Type: "aws_instance",
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

	if len(tfFile.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(tfFile.Resources))
	}

	resource := tfFile.Resources[0]

	if resource.Name != "web" {
		t.Errorf("expected name web, got %s", resource.Name)
	}

	if resource.Type != "aws_instance" {
		t.Errorf("expected type aws_instance, got %s", resource.Type)
	}

	if resource.Attributes["ami"] != "ami-123" {
		t.Errorf("expected ami attribute to be ami-123")
	}
}
