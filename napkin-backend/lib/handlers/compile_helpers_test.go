package handlers

import (
	"napkin-backend/compiler"
	"napkin-backend/graph"
	"strings"
	"testing"
)

func TestIntentGraphToIR_StripsNonWhitelistSpecAndSlugs(t *testing.T) {
	raw := []byte(`{
		"nodes": {
			"resource": [
				{
					"id": "1777593978834",
					"spec": {
						"label": "EC2 Instance",
						"class": "resource",
						"type": "aws_instance"
					},
					"attributes": { "ami": "ami-custom" }
				}
			]
		},
		"edges": []
	}`)

	ig := graph.NewIntentGraph()
	if err := ig.FromJSON(raw); err != nil {
		t.Fatal(err)
	}
	if err := ig.Normalize(); err != nil {
		t.Fatal(err)
	}

	ir, err := IntentGraphToIR(ig)
	if err != nil {
		t.Fatal(err)
	}
	if len(ir.Nodes) != 1 {
		t.Fatalf("nodes: got %d", len(ir.Nodes))
	}
	n := ir.Nodes[0]
	if n.LocalName != "ec2_instance" {
		t.Fatalf("LocalName=%q want ec2_instance", n.LocalName)
	}
	if n.Attributes["ami"] != "ami-custom" {
		t.Fatalf("user ami override missing: %#v", n.Attributes)
	}
	if n.Attributes["label"] != "" {
		t.Fatalf("label leaked into attributes")
	}
}

func TestIntentGraphToIR_DropsMetaKeysFromNodeAttributes(t *testing.T) {
	raw := []byte(`{
		"nodes": {
			"resource": [
				{
					"id": "n1",
					"spec": {
						"label": "EC2 Instance",
						"class": "resource",
						"type": "aws_instance"
					},
					"attributes": {
						"label": "oops",
						"type": "oops_type",
						"ami": "ami-999"
					}
				}
			]
		},
		"edges": []
	}`)

	ig := graph.NewIntentGraph()
	if err := ig.FromJSON(raw); err != nil {
		t.Fatal(err)
	}
	if err := ig.Normalize(); err != nil {
		t.Fatal(err)
	}
	ir, err := IntentGraphToIR(ig)
	if err != nil {
		t.Fatal(err)
	}
	n := ir.Nodes[0]
	if n.Attributes["label"] != "" || n.Attributes["type"] != "" {
		t.Fatalf("meta keys leaked: %#v", n.Attributes)
	}
	if n.Attributes["ami"] != "ami-999" {
		t.Fatalf("want ami-999, got %#v", n.Attributes)
	}
}

func TestIntentGraphToIR_DBToEC2EdgeAndDefaults(t *testing.T) {
	raw := []byte(`{
		"nodes": {
			"resource": [
				{
					"id": "ec2id",
					"spec": {
						"label": "EC2 Instance",
						"class": "resource",
						"type": "aws_instance"
					}
				},
				{
					"id": "dbid",
					"spec": {
						"label": "Database",
						"class": "resource",
						"type": "aws_db_instance"
					}
				}
			]
		},
		"edges": [
			{
				"source": { "node": "resource", "id": "ec2id", "port": "dataSource" },
				"target": { "node": "resource", "id": "dbid", "port": "connection" },
				"type": "data-flow"
			}
		]
	}`)

	ig := graph.NewIntentGraph()
	if err := ig.FromJSON(raw); err != nil {
		t.Fatal(err)
	}
	if err := ig.Normalize(); err != nil {
		t.Fatal(err)
	}

	ir, err := IntentGraphToIR(ig)
	if err != nil {
		t.Fatal(err)
	}
	if len(ir.Edges) != 1 {
		t.Fatalf("edges: got %d", len(ir.Edges))
	}
	if ir.Edges[0].FromID != "ec2id" || ir.Edges[0].ToID != "dbid" {
		t.Fatalf("edge endpoints: %+v", ir.Edges[0])
	}
	if ir.Edges[0].SourcePort != "dataSource" || ir.Edges[0].TargetPort != "connection" {
		t.Fatalf("ports preserved? %+v", ir.Edges[0])
	}

	var db *compiler.GraphNode
	for i := range ir.Nodes {
		if ir.Nodes[i].Type == "aws_db_instance" {
			db = &ir.Nodes[i]
			break
		}
	}
	if db == nil {
		t.Fatal("db node not found")
	}
	if db.LocalName != "database" {
		t.Fatalf("db LocalName=%q", db.LocalName)
	}
	if db.Kind != "database" {
		t.Fatalf("db Kind=%q want database", db.Kind)
	}
	if db.ExprAttributes["allocated_storage"] != "20" {
		t.Fatalf("db defaults: %#v", db.ExprAttributes)
	}
	if db.ExprAttributes["password"] != "var.db_master_password" {
		t.Fatalf("db password should use sensitive variable, got %#v", db.ExprAttributes)
	}
}

func TestIntentGraphToIR_RegionFromJSON(t *testing.T) {
	raw := []byte(`{
		"region": "eu-west-1",
		"nodes": {
			"resource": [
				{
					"id": "n1",
					"spec": { "label": "EC2 Instance", "class": "resource", "type": "aws_instance" }
				}
			]
		},
		"edges": []
	}`)
	ig := graph.NewIntentGraph()
	if err := ig.FromJSON(raw); err != nil {
		t.Fatal(err)
	}
	if err := ig.Normalize(); err != nil {
		t.Fatal(err)
	}
	ir, err := IntentGraphToIR(ig)
	if err != nil {
		t.Fatal(err)
	}
	if ir.Region != "eu-west-1" {
		t.Fatalf("Region=%q want eu-west-1", ir.Region)
	}
}

func TestCompileProducesDependsOn(t *testing.T) {
	raw := []byte(`{
		"nodes": {
			"resource": [
				{
					"id": "ec2id",
					"spec": {
						"label": "EC2 Instance",
						"class": "resource",
						"type": "aws_instance"
					}
				},
				{
					"id": "dbid",
					"spec": {
						"label": "Database",
						"class": "resource",
						"type": "aws_db_instance"
					}
				}
			]
		},
		"edges": [
			{
				"source": { "node": "resource", "id": "ec2id", "port": "dataSource" },
				"target": { "node": "resource", "id": "dbid", "port": "connection" },
				"type": "data-flow"
			}
		]
	}`)

	ig := graph.NewIntentGraph()
	if err := ig.FromJSON(raw); err != nil {
		t.Fatal(err)
	}
	if err := ig.Normalize(); err != nil {
		t.Fatal(err)
	}
	ir, err := IntentGraphToIR(ig)
	if err != nil {
		t.Fatal(err)
	}

	target := &compiler.TerraformTarget{}
	tfFile, err := target.Compile(ir)
	if err != nil {
		t.Fatal(err)
	}
	out := tfFile.String()
	if strings.Contains(out, `depends_on = [aws_db_instance.database]`) {
		t.Fatalf("unexpected explicit depends_on when user_data references DB (redundant and can cycle):\n%s", out)
	}
	if !strings.Contains(out, `user_data = <<-EOT`) {
		t.Fatalf("missing user_data heredoc:\n%s", out)
	}
	if !strings.Contains(out, `DATABASE_HOST=${aws_db_instance.database.address}`) {
		t.Fatalf("missing DATABASE_HOST ref:\n%s", out)
	}
	if !strings.Contains(out, `DATABASE_PORT=${aws_db_instance.database.port}`) {
		t.Fatalf("missing DATABASE_PORT ref:\n%s", out)
	}
	if !strings.Contains(out, `DATABASE_ENDPOINT=${aws_db_instance.database.endpoint}`) {
		t.Fatalf("missing DATABASE_ENDPOINT ref:\n%s", out)
	}
}

func TestCompile_RejectsEdgeWithoutPorts(t *testing.T) {
	raw := []byte(`{
		"nodes": {
			"resource": [
				{
					"id": "ec2id",
					"spec": {
						"label": "EC2 Instance",
						"class": "resource",
						"type": "aws_instance"
					}
				},
				{
					"id": "dbid",
					"spec": {
						"label": "Database",
						"class": "resource",
						"type": "aws_db_instance"
					}
				}
			]
		},
		"edges": [
			{
				"source": { "node": "resource", "id": "dbid" },
				"target": { "node": "resource", "id": "ec2id" },
				"type": "data-flow"
			}
		]
	}`)

	ig := graph.NewIntentGraph()
	if err := ig.FromJSON(raw); err != nil {
		t.Fatal(err)
	}
	if err := ig.Normalize(); err == nil {
		t.Fatal("expected validation error: edge without ports must be rejected")
	}
}

func TestCompileEC2ToDB_ReverseNetworkEdgeDoesNotDependOnEC2(t *testing.T) {
	raw := []byte(`{
		"nodes": {
			"resource": [
				{
					"id": "ec2id",
					"spec": {
						"label": "EC2 Instance",
						"class": "resource",
						"type": "aws_instance"
					}
				},
				{
					"id": "dbid",
					"spec": {
						"label": "Database",
						"class": "resource",
						"type": "aws_db_instance"
					}
				}
			]
		},
		"edges": [
			{
				"source": { "node": "resource", "id": "ec2id", "port": "outboundTraffic" },
				"target": { "node": "resource", "id": "dbid", "port": "inboundTraffic" },
				"type": "data-flow"
			}
		]
	}`)

	ig := graph.NewIntentGraph()
	if err := ig.FromJSON(raw); err != nil {
		t.Fatal(err)
	}
	if err := ig.Normalize(); err != nil {
		t.Fatal(err)
	}
	ir, err := IntentGraphToIR(ig)
	if err != nil {
		t.Fatal(err)
	}

	target := &compiler.TerraformTarget{}
	tfFile, err := target.Compile(ir)
	if err != nil {
		t.Fatal(err)
	}
	out := tfFile.String()
	if strings.Contains(out, `depends_on = [aws_instance.ec2_instance]`) {
		t.Fatalf("unexpected RDS depends_on EC2 for network edge (cycles with data-source wiring):\n%s", out)
	}
}

func TestIntentGraphToIR_IAMRoleGetsAssumeRolePolicyExpr(t *testing.T) {
	raw := []byte(`{
		"nodes": {
			"resource": [
				{
					"id": "role1",
					"kind": "iamRole",
					"spec": {
						"label": "IAM Role",
						"class": "resource",
						"type": "aws_iam_role"
					}
				}
			]
		},
		"edges": []
	}`)

	ig := graph.NewIntentGraph()
	if err := ig.FromJSON(raw); err != nil {
		t.Fatal(err)
	}
	if err := ig.Normalize(); err != nil {
		t.Fatal(err)
	}
	ir, err := IntentGraphToIR(ig)
	if err != nil {
		t.Fatal(err)
	}
	var role *compiler.GraphNode
	for i := range ir.Nodes {
		if ir.Nodes[i].Type == "aws_iam_role" {
			role = &ir.Nodes[i]
			break
		}
	}
	if role == nil {
		t.Fatal("aws_iam_role node missing from IR")
	}
	if role.ExprAttributes["assume_role_policy"] == "" {
		t.Fatalf("expect assume_role_policy from KindIAMRole.ExprDefaults, got %#v", role.ExprAttributes)
	}
}
