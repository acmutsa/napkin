package compiler

import (
	"strings"
	"testing"
)

func TestTerraformTargetCompile(t *testing.T) {
	ir := IR{
		Region: "us-east-1",
		Nodes: []GraphNode{
			{
				ID:        "canvas-web-id",
				LocalName: "web",
				Class:     ClassResource,
				Type:      "aws_instance",
				Attributes: map[string]string{
					"ami":           "ami-custom",
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

	if resourceBlock.Attributes["ami"] != "ami-custom" {
		t.Errorf("expected ami attribute to be ami-custom")
	}

	out := tfFile.String()
	if !strings.Contains(out, `provider "aws"`) {
		t.Errorf("expected provider aws block")
	}
	if !strings.Contains(out, `region = var.aws_region`) {
		t.Errorf("expected provider region from variable")
	}
	if !strings.Contains(out, `variable "aws_region"`) {
		t.Errorf("expected aws_region variable")
	}
	if !strings.Contains(out, `default = "us-east-1"`) {
		t.Errorf("expected default region in variable block")
	}
	if !strings.Contains(out, `resource "aws_vpc"`) {
		t.Errorf("expected napkin VPC scaffold")
	}
	if !strings.Contains(out, `data "aws_ami"`) {
		t.Errorf("expected AMI data source")
	}
}
