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

func TestCompile_OmitsNapkinVPCWhenCanvasSuppliesVPCAndSubnet(t *testing.T) {
	ir := IR{
		Region: "us-east-1",
		Nodes: []GraphNode{
			{
				ID:        "vpc1",
				LocalName: "canvas_vpc",
				Kind:      "vpc",
				Class:     ClassResource,
				Type:      "aws_vpc",
				Attributes: map[string]string{
					"cidr_block": "10.99.0.0/16",
				},
			},
			{
				ID:        "sn1",
				LocalName: "canvas_sn",
				Kind:      "subnet",
				Class:     ClassResource,
				Type:      "aws_subnet",
				Attributes: map[string]string{
					"cidr_block": "10.99.1.0/24",
				},
			},
			{
				ID:        "web",
				LocalName: "web",
				Kind:      "compute",
				Class:     ClassResource,
				Type:      "aws_instance",
				Attributes: map[string]string{
					"ami":           "ami-custom",
					"instance_type": "t2.micro",
				},
			},
		},
		Edges: []DirectedEdge{
			{FromID: "vpc1", ToID: "sn1", SourcePort: "network", TargetPort: "vpc"},
			{FromID: "sn1", ToID: "web", SourcePort: "placement", TargetPort: "subnet"},
		},
	}

	out, err := (&TerraformTarget{}).Compile(ir)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if strings.Contains(s, `resource "aws_vpc" "napkin_vpc"`) {
		t.Fatalf("napkin_vpc should not emit when canvas provides aws_vpc + aws_subnet:\n%s", s)
	}
	if !strings.Contains(s, `resource "aws_vpc" "canvas_vpc"`) {
		t.Fatalf("missing canvas VPC resource:\n%s", s)
	}
	if !strings.Contains(s, `subnet_id = aws_subnet.canvas_sn.id`) {
		t.Fatalf("EC2 should attach to canvas subnet via edge:\n%s", s)
	}
	if !strings.Contains(s, `vpc_id = aws_vpc.canvas_vpc.id`) {
		t.Fatalf("subnet should reference canvas VPC via edge:\n%s", s)
	}
}

func TestCompile_RejectsUnsupportedConnection(t *testing.T) {
	// Same-type ports (network -> network) but no compiler wiring registered:
	// vpc.network -> compute.subnet is meaningless (compute expects subnet.placement).
	ir := IR{
		Region: "us-east-1",
		Nodes: []GraphNode{
			{ID: "vpc1", LocalName: "vpc", Kind: "vpc", Class: ClassResource, Type: "aws_vpc"},
			{ID: "ec2", LocalName: "web", Kind: "compute", Class: ClassResource, Type: "aws_instance"},
		},
		Edges: []DirectedEdge{
			{FromID: "vpc1", ToID: "ec2", SourcePort: "network", TargetPort: "subnet"},
		},
	}
	if _, err := (&TerraformTarget{}).Compile(ir); err == nil {
		t.Fatal("expected compile error for unsupported (vpc.network -> compute.subnet) edge")
	}
}

func TestCompile_LBToEC2EdgeWiresTargetGroup(t *testing.T) {
	ir := IR{
		Region: "us-east-1",
		Nodes: []GraphNode{
			{ID: "lb", LocalName: "alb", Kind: "loadBalancer", Class: ClassResource, Type: "aws_lb"},
			{ID: "ec2", LocalName: "web", Kind: "compute", Class: ClassResource, Type: "aws_instance",
				Attributes: map[string]string{"ami": "ami-custom"}},
		},
		Edges: []DirectedEdge{
			{FromID: "lb", ToID: "ec2", SourcePort: "forward", TargetPort: "inboundTraffic"},
		},
	}
	out, err := (&TerraformTarget{}).Compile(ir)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, `resource "aws_lb_target_group" "tg_alb"`) {
		t.Fatalf("missing target group:\n%s", s)
	}
	if !strings.Contains(s, `resource "aws_lb_listener" "lis_alb"`) {
		t.Fatalf("missing listener:\n%s", s)
	}
	if !strings.Contains(s, `target_id = aws_instance.web.id`) {
		t.Fatalf("missing target attachment to EC2:\n%s", s)
	}
}

func TestCompile_IAMToLambdaEdgeSetsRole(t *testing.T) {
	ir := IR{
		Region: "us-east-1",
		Nodes: []GraphNode{
			{ID: "role1", LocalName: "lambda_role", Kind: "iamRole", Class: ClassResource, Type: "aws_iam_role"},
			{ID: "fn", LocalName: "fn", Kind: "lambda", Class: ClassResource, Type: "aws_lambda_function"},
		},
		Edges: []DirectedEdge{
			{FromID: "role1", ToID: "fn", SourcePort: "role", TargetPort: "executionRole"},
		},
	}
	out, err := (&TerraformTarget{}).Compile(ir)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, `role = aws_iam_role.lambda_role.arn`) {
		t.Fatalf("expected lambda.role wiring from edge:\n%s", s)
	}
}

func TestCompile_QueueToLambdaEmitsEventSourceMapping(t *testing.T) {
	ir := IR{
		Region: "us-east-1",
		Nodes: []GraphNode{
			{ID: "q", LocalName: "events", Kind: "sqsQueue", Class: ClassResource, Type: "aws_sqs_queue"},
			{ID: "fn", LocalName: "fn", Kind: "lambda", Class: ClassResource, Type: "aws_lambda_function"},
		},
		Edges: []DirectedEdge{
			{FromID: "q", ToID: "fn", SourcePort: "consumer", TargetPort: "eventSource"},
		},
	}
	out, err := (&TerraformTarget{}).Compile(ir)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, `resource "aws_lambda_event_source_mapping" "fn_events_evt"`) {
		t.Fatalf("expected event source mapping for SQS->Lambda edge:\n%s", s)
	}
	if !strings.Contains(s, `event_source_arn = aws_sqs_queue.events.arn`) {
		t.Fatalf("expected event_source_arn pointing at queue:\n%s", s)
	}
}

func TestCompile_IAMToEC2EmitsInstanceProfile(t *testing.T) {
	ir := IR{
		Region: "us-east-1",
		Nodes: []GraphNode{
			{ID: "role1", LocalName: "ec2_role", Kind: "iamRole", Class: ClassResource, Type: "aws_iam_role"},
			{ID: "ec2", LocalName: "web", Kind: "compute", Class: ClassResource, Type: "aws_instance",
				Attributes: map[string]string{"ami": "ami-custom"}},
		},
		Edges: []DirectedEdge{
			{FromID: "role1", ToID: "ec2", SourcePort: "role", TargetPort: "instanceRole"},
		},
	}
	out, err := (&TerraformTarget{}).Compile(ir)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, `resource "aws_iam_instance_profile" "ec2_role_profile"`) {
		t.Fatalf("expected instance profile resource:\n%s", s)
	}
	if !strings.Contains(s, `iam_instance_profile = aws_iam_instance_profile.ec2_role_profile.name`) {
		t.Fatalf("expected EC2 to reference instance profile:\n%s", s)
	}
}

func TestCompile_SubnetToDatabaseEmitsSubnetGroup(t *testing.T) {
	ir := IR{
		Region: "us-east-1",
		Nodes: []GraphNode{
			{ID: "sn1", LocalName: "sn_a", Kind: "subnet", Class: ClassResource, Type: "aws_subnet"},
			{ID: "sn2", LocalName: "sn_b", Kind: "subnet", Class: ClassResource, Type: "aws_subnet"},
			{ID: "db", LocalName: "primary", Kind: "database", Class: ClassResource, Type: "aws_db_instance"},
		},
		Edges: []DirectedEdge{
			{FromID: "sn1", ToID: "db", SourcePort: "placement", TargetPort: "subnet"},
			{FromID: "sn2", ToID: "db", SourcePort: "placement", TargetPort: "subnet"},
		},
	}
	out, err := (&TerraformTarget{}).Compile(ir)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, `resource "aws_db_subnet_group" "primary_sg"`) {
		t.Fatalf("expected db subnet group:\n%s", s)
	}
	if !strings.Contains(s, `db_subnet_group_name = aws_db_subnet_group.primary_sg.name`) {
		t.Fatalf("expected db to reference its subnet group:\n%s", s)
	}
}

func TestCompile_SGToEC2AppendsSecurityGroup(t *testing.T) {
	ir := IR{
		Region: "us-east-1",
		Nodes: []GraphNode{
			{ID: "sg", LocalName: "app_sg", Kind: "securityGroup", Class: ClassResource, Type: "aws_security_group"},
			{ID: "ec2", LocalName: "web", Kind: "compute", Class: ClassResource, Type: "aws_instance",
				Attributes: map[string]string{"ami": "ami-custom"}},
		},
		Edges: []DirectedEdge{
			{FromID: "sg", ToID: "ec2", SourcePort: "attachment", TargetPort: "securityGroup"},
		},
	}
	out, err := (&TerraformTarget{}).Compile(ir)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, `vpc_security_group_ids = [aws_security_group.app_sg.id]`) {
		t.Fatalf("expected EC2 to bind to canvas SG:\n%s", s)
	}
}

func TestCompile_EC2ToDBDataSourceEdgeInjectsDBEnv(t *testing.T) {
	ir := IR{
		Region: "us-east-1",
		Nodes: []GraphNode{
			{ID: "ec2", LocalName: "web", Kind: "compute", Class: ClassResource, Type: "aws_instance",
				Attributes: map[string]string{"ami": "ami-custom"}},
			{ID: "db", LocalName: "database", Kind: "database", Class: ClassResource, Type: "aws_db_instance"},
		},
		Edges: []DirectedEdge{
			{FromID: "ec2", ToID: "db", SourcePort: "dataSource", TargetPort: "connection"},
		},
	}
	out, err := (&TerraformTarget{}).Compile(ir)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, `depends_on = [aws_db_instance.database]`) {
		t.Fatalf("missing depends_on on EC2 for dataSource edge:\n%s", s)
	}
	if !strings.Contains(s, `DATABASE_HOST=${aws_db_instance.database.address}`) {
		t.Fatalf("missing DATABASE_HOST injection:\n%s", s)
	}
}
