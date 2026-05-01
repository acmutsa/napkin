package compiler

import (
	"strings"
	"testing"
)

func TestTerraformTargetCompile(t *testing.T) {
	ir := IR{
		Nodes: []GraphNode{
			{
				ID:        "canvas-web-id",
				LocalName: "web",
				Class:     ClassResource,
				Type:      "aws_instance",
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

	var resourceBlock *TFBlock
	for i := range tfFile.Block {
		if tfFile.Block[i].Class == "resource" && len(tfFile.Block[i].Labels) > 0 && tfFile.Block[i].Labels[0] == "aws_instance" {
			resourceBlock = &tfFile.Block[i]
			break
		}
	}
	if resourceBlock == nil {
		t.Fatalf("no aws_instance resource block in %#v", tfFile.Block)
	}

	if len(resourceBlock.Labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(resourceBlock.Labels))
	}

	if resourceBlock.Labels[0] != "aws_instance" {
		t.Errorf("expected type aws_instance, got %s", resourceBlock.Labels[0])
	}

	if resourceBlock.Labels[1] != "web" {
		t.Errorf("expected name web, got %s", resourceBlock.Labels[1])
	}

	if resourceBlock.Attributes["ami"] != "ami-123" {
		t.Errorf("expected ami attribute to be ami-123")
	}

	out := tfFile.String()
	if !strings.Contains(out, `provider "aws"`) {
		t.Errorf("expected provider aws block")
	}
	if !strings.Contains(out, `region = "us-east-1"`) {
		t.Errorf("expected default region")
	}
}
