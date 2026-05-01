package compiler

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

const (
	napkinVPC      = "napkin_vpc"
	napkinSubnetA  = "napkin_public_a"
	napkinSubnetB  = "napkin_public_b"
	napkinIGW      = "napkin_igw"
	napkinPublicRT = "napkin_public_rt"
	napkinAZData   = "available"
	napkinAMIData  = "napkin_al2023"
	napkinSGALB    = "napkin_alb"
	napkinSGEC2    = "napkin_ec2"
)

type TerraformTarget struct {
}

func napkinBanner(line string) TFBlock {
	return TFBlock{RawLine: line}
}

func effectiveRegion(ir IR) string {
	r := strings.TrimSpace(ir.Region)
	if r == "" {
		return "us-east-1"
	}
	return r
}

func (t *TerraformTarget) Compile(ir IR) (*TFFile, error) {
	region := effectiveRegion(ir)

	tfFile := &TFFile{}

	tfFile.Block = append(tfFile.Block,
		napkinBanner("# --- napkin: terraform ---"),
		terraformRequiredProvidersBlock(),
	)

	tfFile.Block = append(tfFile.Block,
		napkinBanner("# --- napkin: variables ---"),
		variableBlock("aws_region", map[string]string{"default": region}, map[string]string{"type": "string"}),
		variableBlock("db_master_password", nil, map[string]string{"type": "string", "sensitive": "true"}),
	)

	tfFile.Block = append(tfFile.Block,
		napkinBanner("# --- napkin: provider ---"),
		TFBlock{
			Class:          "provider",
			Labels:         []string{"aws"},
			ExprAttributes: map[string]string{"region": "var.aws_region"},
		},
	)

	tfFile.Block = append(tfFile.Block,
		napkinBanner("# --- napkin: data & network ---"),
	)
	tfFile.Block = append(tfFile.Block, napkinDataBlocks()...)
	tfFile.Block = append(tfFile.Block, napkinNetworkBlocks()...)

	tfFile.Block = append(tfFile.Block,
		napkinBanner("# --- napkin: security groups ---"),
	)
	tfFile.Block = append(tfFile.Block, napkinSecurityGroupBlocks()...)

	idToNode := make(map[string]GraphNode, len(ir.Nodes))
	for _, n := range ir.Nodes {
		idToNode[n.ID] = n
	}

	dependsRefs := make(map[string][]string)
	rdsRefsByEC2 := make(map[string][]string)

	for _, e := range ir.Edges {
		from, ok1 := idToNode[e.FromID]
		to, ok2 := idToNode[e.ToID]
		if !ok1 || !ok2 {
			continue
		}
		db, ec2, ok := linkedRDSAndEC2(from, to)
		if !ok || !edgeAllowsRDSLink(e, from, to) {
			continue
		}
		ref := fmt.Sprintf("aws_db_instance.%s", db.LocalName)
		dependsRefs[ec2.ID] = appendUniqueRef(dependsRefs[ec2.ID], ref)
		rdsRefsByEC2[ec2.ID] = appendUniqueRef(rdsRefsByEC2[ec2.ID], db.LocalName)
	}

	lbToEC2 := napkinLBTargets(ir.Edges, idToNode)

	tfFile.Block = append(tfFile.Block, napkinBanner("# --- napkin: canvas resources ---"))

	var lbNodes []GraphNode
	for _, node := range ir.Nodes {
		block := TFBlock{
			Class:          string(node.Class),
			Attributes:     make(map[string]string),
			ExprAttributes: make(map[string]string),
		}

		maps.Copy(block.Attributes, node.Attributes)
		if node.ExprAttributes != nil {
			maps.Copy(block.ExprAttributes, node.ExprAttributes)
		}

		if refs := dependsRefs[node.ID]; len(refs) > 0 {
			slices.Sort(refs)
			block.ExprAttributes["depends_on"] = "[" + strings.Join(refs, ", ") + "]"
		}

		if node.Type == "aws_instance" {
			slugs := rdsRefsByEC2[node.ID]
			slices.Sort(slugs)
			if !hasUserData(block) {
				block.ExprAttributes["user_data"] = napkinEC2UserData(slugs)
			}
			napkinEnhanceEC2Instance(&block)
		}

		if node.Type == "aws_lb" {
			lbNodes = append(lbNodes, node)
			napkinEnhanceALB(&block)
		}

		local := node.LocalName
		if local == "" {
			local = node.ID
		}

		switch node.Class {
		case ClassResource:
			block.Labels = []string{node.Type, local}
		case ClassData:
			block.Labels = []string{node.Type, local}
		case ClassProvider:
			block.Labels = []string{node.Type}
		case ClassModule:
			block.Labels = []string{local}
		default:
			return nil, fmt.Errorf("unsupported node class: %s", node.Class)
		}

		tfFile.Block = append(tfFile.Block, block)
	}

	if len(lbNodes) > 0 {
		tfFile.Block = append(tfFile.Block, napkinBanner("# --- napkin: ALB listeners & attachments ---"))
	}
	tgCounter := 1
	for _, lb := range lbNodes {
		targets := lbToEC2[lb.ID]
		tgPrefix := fmt.Sprintf("nk%02d", tgCounter)
		tgCounter++
		tgName := "tg_" + lb.LocalName
		listenerName := "lis_" + lb.LocalName

		tfFile.Block = append(tfFile.Block, napkinLBTargetGroupBlock(tgName, tgPrefix))
		tfFile.Block = append(tfFile.Block, napkinLBListenerBlock(listenerName, lb.LocalName, tgName))

		for _, ec2 := range targets {
			attName := fmt.Sprintf("att_%s_%s", lb.LocalName, ec2.LocalName)
			tfFile.Block = append(tfFile.Block, napkinLBTargetAttachmentBlock(attName, tgName, ec2.LocalName))
		}
	}

	if len(lbNodes) > 0 {
		tfFile.Block = append(tfFile.Block, napkinBanner("# --- napkin: outputs ---"))
		first := lbNodes[0]
		tfFile.Block = append(tfFile.Block, TFBlock{
			Class:  "output",
			Labels: []string{"alb_dns_name"},
			ExprAttributes: map[string]string{
				"value":       fmt.Sprintf("aws_lb.%s.dns_name", first.LocalName),
				"description": `"Public DNS name of the first Application Load Balancer"`,
			},
		})
	}

	return tfFile, nil
}

func terraformRequiredProvidersBlock() TFBlock {
	return TFBlock{
		Class: "terraform",
		Blocks: []TFBlock{
			{
				Class: "required_providers",
				Blocks: []TFBlock{
					{
						Class: "aws =",
						Attributes: map[string]string{
							"source":  "hashicorp/aws",
							"version": "~> 5.0",
						},
					},
				},
			},
		},
	}
}

func variableBlock(name string, attrs map[string]string, expr map[string]string) TFBlock {
	a := map[string]string{}
	if attrs != nil {
		maps.Copy(a, attrs)
	}
	e := map[string]string{}
	if expr != nil {
		maps.Copy(e, expr)
	}
	return TFBlock{
		Class:          "variable",
		Labels:         []string{name},
		Attributes:     a,
		ExprAttributes: e,
	}
}

func napkinDataBlocks() []TFBlock {
	return []TFBlock{
		{
			Class:      "data",
			Labels:     []string{"aws_availability_zones", napkinAZData},
			Attributes: map[string]string{"state": "available"},
		},
		{
			Class:  "data",
			Labels: []string{"aws_ami", napkinAMIData},
			ExprAttributes: map[string]string{
				"most_recent": "true",
				"owners":      `["amazon"]`,
			},
			Blocks: []TFBlock{
				{
					Class:          "filter",
					Attributes:     map[string]string{"name": "name"},
					ExprAttributes: map[string]string{"values": `["al2023-ami-*-x86_64"]`},
				},
				{
					Class:          "filter",
					Attributes:     map[string]string{"name": "virtualization-type"},
					ExprAttributes: map[string]string{"values": `["hvm"]`},
				},
			},
		},
	}
}

func napkinNetworkBlocks() []TFBlock {
	vpcID := "aws_vpc." + napkinVPC + ".id"
	subnetAID := "aws_subnet." + napkinSubnetA + ".id"
	subnetBID := "aws_subnet." + napkinSubnetB + ".id"
	igwID := "aws_internet_gateway." + napkinIGW + ".id"
	rtID := "aws_route_table." + napkinPublicRT + ".id"
	azNames := "data.aws_availability_zones." + napkinAZData + ".names"

	return []TFBlock{
		{
			Class:      "resource",
			Labels:     []string{"aws_vpc", napkinVPC},
			Attributes: map[string]string{"cidr_block": "10.0.0.0/16"},
			ExprAttributes: map[string]string{
				"enable_dns_hostnames": "true",
				"enable_dns_support":   "true",
			},
		},
		{
			Class:  "resource",
			Labels: []string{"aws_subnet", napkinSubnetA},
			ExprAttributes: map[string]string{
				"vpc_id":                  vpcID,
				"availability_zone":       azNames + "[0]",
				"map_public_ip_on_launch": "true",
			},
			Attributes: map[string]string{"cidr_block": "10.0.1.0/24"},
		},
		{
			Class:  "resource",
			Labels: []string{"aws_subnet", napkinSubnetB},
			ExprAttributes: map[string]string{
				"vpc_id":                  vpcID,
				"availability_zone":       azNames + "[1]",
				"map_public_ip_on_launch": "true",
			},
			Attributes: map[string]string{"cidr_block": "10.0.2.0/24"},
		},
		{
			Class:          "resource",
			Labels:         []string{"aws_internet_gateway", napkinIGW},
			ExprAttributes: map[string]string{"vpc_id": vpcID},
		},
		{
			Class:          "resource",
			Labels:         []string{"aws_route_table", napkinPublicRT},
			ExprAttributes: map[string]string{"vpc_id": vpcID},
			Blocks: []TFBlock{
				{
					Class:          "route",
					Attributes:     map[string]string{"cidr_block": "0.0.0.0/0"},
					ExprAttributes: map[string]string{"gateway_id": igwID},
				},
			},
		},
		{
			Class:  "resource",
			Labels: []string{"aws_route_table_association", "napkin_public_a_assoc"},
			ExprAttributes: map[string]string{
				"subnet_id":      subnetAID,
				"route_table_id": rtID,
			},
		},
		{
			Class:  "resource",
			Labels: []string{"aws_route_table_association", "napkin_public_b_assoc"},
			ExprAttributes: map[string]string{
				"subnet_id":      subnetBID,
				"route_table_id": rtID,
			},
		},
	}
}

func napkinSecurityGroupBlocks() []TFBlock {
	vpcID := "aws_vpc." + napkinVPC + ".id"
	albSG := "aws_security_group." + napkinSGALB + ".id"

	return []TFBlock{
		{
			Class:          "resource",
			Labels:         []string{"aws_security_group", napkinSGALB},
			Attributes:     map[string]string{"name": "napkin-alb"},
			ExprAttributes: map[string]string{"vpc_id": vpcID},
			Blocks: []TFBlock{
				{
					Class: "ingress",
					ExprAttributes: map[string]string{
						"description": `"HTTP from anywhere (demo)"`,
						"from_port":   "80",
						"to_port":     "80",
						"protocol":    `"tcp"`,
						"cidr_blocks": `["0.0.0.0/0"]`,
					},
				},
				{
					Class: "egress",
					ExprAttributes: map[string]string{
						"from_port":   "0",
						"to_port":     "0",
						"protocol":    `"-1"`,
						"cidr_blocks": `["0.0.0.0/0"]`,
					},
				},
			},
		},
		{
			Class:          "resource",
			Labels:         []string{"aws_security_group", napkinSGEC2},
			Attributes:     map[string]string{"name": "napkin-ec2"},
			ExprAttributes: map[string]string{"vpc_id": vpcID},
			Blocks: []TFBlock{
				{
					Class: "ingress",
					ExprAttributes: map[string]string{
						"description":     `"HTTP from ALB"`,
						"from_port":       "80",
						"to_port":         "80",
						"protocol":        `"tcp"`,
						"security_groups": "[" + albSG + "]",
					},
				},
				{
					Class: "egress",
					ExprAttributes: map[string]string{
						"from_port":   "0",
						"to_port":     "0",
						"protocol":    `"-1"`,
						"cidr_blocks": `["0.0.0.0/0"]`,
					},
				},
			},
		},
	}
}

func napkinEnhanceEC2Instance(block *TFBlock) {
	subnetExpr := "aws_subnet." + napkinSubnetA + ".id"
	sgExpr := "[" + "aws_security_group." + napkinSGEC2 + ".id" + "]"

	block.ExprAttributes["subnet_id"] = subnetExpr
	block.ExprAttributes["vpc_security_group_ids"] = sgExpr

	if block.Attributes["ami"] != "" {
		return
	}
	block.ExprAttributes["ami"] = "data.aws_ami." + napkinAMIData + ".id"
}

func napkinEnhanceALB(block *TFBlock) {
	subnets := "[aws_subnet." + napkinSubnetA + ".id, aws_subnet." + napkinSubnetB + ".id]"
	sgs := "[" + "aws_security_group." + napkinSGALB + ".id" + "]"
	block.ExprAttributes["subnets"] = subnets
	block.ExprAttributes["security_groups"] = sgs
	block.ExprAttributes["internal"] = "false"
}

func napkinEC2UserData(rdsSlugs []string) string {
	var b strings.Builder
	b.WriteString("<<-EOT\n#!/bin/bash\nset -euo pipefail\n")
	b.WriteString("# Napkin Phase 1: serve HTTP on :80 for ALB health checks\n")
	b.WriteString("if command -v dnf >/dev/null 2>&1; then dnf install -y nginx; ")
	b.WriteString("elif command -v yum >/dev/null 2>&1; then yum install -y nginx; ")
	b.WriteString("else apt-get update && apt-get install -y nginx; fi\n")
	b.WriteString("systemctl enable nginx\nsystemctl start nginx\n")

	for i, slug := range rdsSlugs {
		ref := fmt.Sprintf("aws_db_instance.%s", slug)
		if len(rdsSlugs) == 1 {
			b.WriteString(fmt.Sprintf("echo \"DATABASE_HOST=${%s.address}\" >> /etc/environment\n", ref))
			b.WriteString(fmt.Sprintf("echo \"DATABASE_PORT=${%s.port}\" >> /etc/environment\n", ref))
			b.WriteString(fmt.Sprintf("echo \"DATABASE_ENDPOINT=${%s.endpoint}\" >> /etc/environment\n", ref))
			continue
		}
		n := i + 1
		b.WriteString(fmt.Sprintf("echo \"DATABASE_%d_HOST=${%s.address}\" >> /etc/environment\n", n, ref))
		b.WriteString(fmt.Sprintf("echo \"DATABASE_%d_PORT=${%s.port}\" >> /etc/environment\n", n, ref))
		b.WriteString(fmt.Sprintf("echo \"DATABASE_%d_ENDPOINT=${%s.endpoint}\" >> /etc/environment\n", n, ref))
	}
	b.WriteString("EOT")
	return b.String()
}

func napkinLBTargets(edges []DirectedEdge, idToNode map[string]GraphNode) map[string][]GraphNode {
	out := make(map[string][]GraphNode)
	for _, e := range edges {
		from := idToNode[e.FromID]
		to := idToNode[e.ToID]
		if from.Type != "aws_lb" || to.Type != "aws_instance" {
			continue
		}
		if !edgeAllowsLBToEC2(e) {
			continue
		}
		out[from.ID] = append(out[from.ID], to)
	}
	return out
}

func edgeAllowsLBToEC2(e DirectedEdge) bool {
	if e.SourcePort != "" && e.SourcePort != "out-network" {
		return false
	}
	if e.TargetPort != "" && e.TargetPort != "in-network" {
		return false
	}
	return true
}

func napkinLBTargetGroupBlock(tfName, namePrefix string) TFBlock {
	return TFBlock{
		Class:      "resource",
		Labels:     []string{"aws_lb_target_group", tfName},
		Attributes: map[string]string{"name_prefix": namePrefix, "protocol": "HTTP"},
		ExprAttributes: map[string]string{
			"port":                 "80",
			"vpc_id":               "aws_vpc." + napkinVPC + ".id",
			"target_type":          `"instance"`,
			"deregistration_delay": "30",
		},
	}
}

func napkinLBListenerBlock(tfName, lbLocal, tgTFName string) TFBlock {
	tgArn := "aws_lb_target_group." + tgTFName + ".arn"
	return TFBlock{
		Class:      "resource",
		Labels:     []string{"aws_lb_listener", tfName},
		Attributes: map[string]string{"protocol": "HTTP"},
		ExprAttributes: map[string]string{
			"load_balancer_arn": "aws_lb." + lbLocal + ".arn",
			"port":              "80",
		},
		Blocks: []TFBlock{
			{
				Class:          "default_action",
				Attributes:     map[string]string{"type": "forward"},
				ExprAttributes: map[string]string{"target_group_arn": tgArn},
			},
		},
	}
}

func napkinLBTargetAttachmentBlock(tfName, tgTFName, ec2Local string) TFBlock {
	return TFBlock{
		Class:  "resource",
		Labels: []string{"aws_lb_target_group_attachment", tfName},
		ExprAttributes: map[string]string{
			"target_group_arn": "aws_lb_target_group." + tgTFName + ".arn",
			"target_id":        "aws_instance." + ec2Local + ".id",
			"port":             "80",
		},
	}
}

func hasUserData(block TFBlock) bool {
	if block.Attributes["user_data"] != "" {
		return true
	}
	if block.ExprAttributes != nil {
		if _, ok := block.ExprAttributes["user_data"]; ok {
			return true
		}
	}
	return false
}

func linkedRDSAndEC2(a, b GraphNode) (db GraphNode, ec2 GraphNode, ok bool) {
	switch {
	case a.Type == "aws_db_instance" && b.Type == "aws_instance":
		return a, b, true
	case a.Type == "aws_instance" && b.Type == "aws_db_instance":
		return b, a, true
	default:
		return GraphNode{}, GraphNode{}, false
	}
}

// edgeAllowsRDSLink accepts edges even when React Flow omits handle ids (common cause of missing compile links).
// Port names follow the typed-port vocabulary in napkin-backend/graph/kinds.go and napkin-app/src/lib/kinds.ts.
func edgeAllowsRDSLink(e DirectedEdge, from, to GraphNode) bool {
	switch {
	case from.Type == "aws_db_instance" && to.Type == "aws_instance":
		// Database out-data -> EC2 in-env (connection injected as env vars) or in-network (reachability).
		if e.SourcePort != "" && e.SourcePort != "out-data" {
			return false
		}
		if e.TargetPort != "" && e.TargetPort != "in-env" && e.TargetPort != "in-network" {
			return false
		}
		return true
	case from.Type == "aws_instance" && to.Type == "aws_db_instance":
		// EC2 out-network/out-data -> Database in-network/in-env.
		if e.SourcePort != "" && e.SourcePort != "out-network" && e.SourcePort != "out-data" {
			return false
		}
		if e.TargetPort != "" && e.TargetPort != "in-network" && e.TargetPort != "in-env" {
			return false
		}
		return true
	default:
		return false
	}
}

func appendUniqueRef(list []string, ref string) []string {
	for _, x := range list {
		if x == ref {
			return list
		}
	}
	return append(list, ref)
}

func (t *TerraformTarget) ToString() string {
	return ""
}

var _ Target = (*TerraformTarget)(nil)
