package compiler

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// networkBindings tells the compiler whether to emit the built-in napkin VPC/subnets
// or attach SG / EC2 / ALB / target groups to VPC + subnets drawn on the canvas.
//
// With the V1 port model these defaults are only used as a *fallback* when a
// canvas resource has no explicit subnet/SG edges. Explicit edges always win.
type networkBindings struct {
	useCanvasNet    bool
	vpcIDExpr       string // Terraform expression, e.g. aws_vpc.foo.id
	subnetAExpr     string // aws_subnet.xxx.id
	subnetBExpr     string // second subnet for ALB (may equal subnetAExpr if only one)
	primaryVPCLocal string // LocalName of first canvas VPC; empty when using napkin network
}

func resolveNetworkBindings(ir IR) networkBindings {
	var vpcs, subnets []GraphNode
	for _, n := range ir.Nodes {
		if n.Class != ClassResource {
			continue
		}
		switch n.Type {
		case "aws_vpc":
			vpcs = append(vpcs, n)
		case "aws_subnet":
			subnets = append(subnets, n)
		}
	}

	if len(vpcs) < 1 || len(subnets) < 1 {
		return networkBindings{
			useCanvasNet: false,
			vpcIDExpr:    "aws_vpc." + napkinVPC + ".id",
			subnetAExpr:  "aws_subnet." + napkinSubnetA + ".id",
			subnetBExpr:  "aws_subnet." + napkinSubnetB + ".id",
		}
	}

	slices.SortFunc(vpcs, func(a, b GraphNode) int {
		return strings.Compare(a.LocalName, b.LocalName)
	})
	slices.SortFunc(subnets, func(a, b GraphNode) int {
		return strings.Compare(a.LocalName, b.LocalName)
	})

	subnetA := "aws_subnet." + subnets[0].LocalName + ".id"
	subnetB := subnetA
	if len(subnets) >= 2 {
		subnetB = "aws_subnet." + subnets[1].LocalName + ".id"
	}

	return networkBindings{
		useCanvasNet:    true,
		vpcIDExpr:       "aws_vpc." + vpcs[0].LocalName + ".id",
		subnetAExpr:     subnetA,
		subnetBExpr:     subnetB,
		primaryVPCLocal: vpcs[0].LocalName,
	}
}

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
	nb := resolveNetworkBindings(ir)
	if !nb.useCanvasNet {
		tfFile.Block = append(tfFile.Block, napkinNetworkBlocks()...)
	}

	tfFile.Block = append(tfFile.Block,
		napkinBanner("# --- napkin: security groups ---"),
	)
	tfFile.Block = append(tfFile.Block, napkinSecurityGroupBlocks(nb.vpcIDExpr)...)

	idToNode := make(map[string]GraphNode, len(ir.Nodes))
	for _, n := range ir.Nodes {
		idToNode[n.ID] = n
	}

	bindings, err := resolveEdges(ir.Edges, idToNode)
	if err != nil {
		return nil, err
	}

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

		applyNodeBindings(&block, node, bindings, nb)

		switch node.Type {
		case "aws_subnet":
			applySubnetBindings(&block, node, bindings, nb)
		case "aws_security_group":
			applySecurityGroupBindings(&block, node, bindings, nb)
		case "aws_instance":
			applyComputeBindings(&block, node, bindings, nb)
		case "aws_lb":
			lbNodes = append(lbNodes, node)
			applyLoadBalancerBindings(&block, node, bindings, nb)
		case "aws_db_instance":
			applyDatabaseBindings(&block, node, bindings)
		case "aws_lambda_function":
			applyLambdaBindings(&block, node, bindings)
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

	if dbSubnetGroups := emitDBSubnetGroups(ir, bindings); len(dbSubnetGroups) > 0 {
		tfFile.Block = append(tfFile.Block, napkinBanner("# --- napkin: db subnet groups ---"))
		tfFile.Block = append(tfFile.Block, dbSubnetGroups...)
	}

	if iamProfiles := emitInstanceProfiles(ir, bindings); len(iamProfiles) > 0 {
		tfFile.Block = append(tfFile.Block, napkinBanner("# --- napkin: iam instance profiles ---"))
		tfFile.Block = append(tfFile.Block, iamProfiles...)
	}

	if eventSourceBlocks := emitLambdaEventSources(ir, bindings); len(eventSourceBlocks) > 0 {
		tfFile.Block = append(tfFile.Block, napkinBanner("# --- napkin: lambda event sources ---"))
		tfFile.Block = append(tfFile.Block, eventSourceBlocks...)
	}

	if len(lbNodes) > 0 {
		tfFile.Block = append(tfFile.Block, napkinBanner("# --- napkin: ALB listeners & attachments ---"))
	}
	tgCounter := 1
	for _, lb := range lbNodes {
		targets := bindings.lbTargets[lb.ID]
		tgPrefix := fmt.Sprintf("nk%02d", tgCounter)
		tgCounter++
		tgName := "tg_" + lb.LocalName
		listenerName := "lis_" + lb.LocalName

		tfFile.Block = append(tfFile.Block, napkinLBTargetGroupBlock(tgName, tgPrefix, lbVPCExpr(lb, bindings, nb)))
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

// applyNodeBindings is shared logic that does not depend on Terraform type.
// (Currently a no-op placeholder; kept so future cross-cutting wiring like
// tags or shared depends_on aggregation has a single insertion point.)
func applyNodeBindings(_ *TFBlock, _ GraphNode, _ *edgeBindings, _ networkBindings) {
}

// applySubnetBindings sets vpc_id from a vpc->subnet edge if present, else
// falls back to the canvas VPC (when the canvas drew a VPC) or napkin's VPC.
func applySubnetBindings(block *TFBlock, node GraphNode, b *edgeBindings, nb networkBindings) {
	if hasVPCID(*block) {
		return
	}
	if vpcLocal, ok := b.vpcOf[node.ID]; ok && vpcLocal != "" {
		block.ExprAttributes["vpc_id"] = "aws_vpc." + vpcLocal + ".id"
		return
	}
	if nb.useCanvasNet && nb.primaryVPCLocal != "" {
		block.ExprAttributes["vpc_id"] = "aws_vpc." + nb.primaryVPCLocal + ".id"
	}
}

// applySecurityGroupBindings sets vpc_id from a vpc->sg edge if present, else
// falls back to napkin's VPC when no canvas VPC was drawn.
func applySecurityGroupBindings(block *TFBlock, node GraphNode, b *edgeBindings, nb networkBindings) {
	if hasVPCID(*block) {
		return
	}
	if vpcLocal, ok := b.vpcOf[node.ID]; ok && vpcLocal != "" {
		block.ExprAttributes["vpc_id"] = "aws_vpc." + vpcLocal + ".id"
		return
	}
	if nb.useCanvasNet && nb.primaryVPCLocal != "" {
		block.ExprAttributes["vpc_id"] = "aws_vpc." + nb.primaryVPCLocal + ".id"
	} else {
		block.ExprAttributes["vpc_id"] = nb.vpcIDExpr
	}
}

// applyComputeBindings wires subnet/security groups/data sources/IAM on EC2.
// Explicit edges win; otherwise we fall back to napkin's defaults so a
// minimal "drop an EC2 on the canvas" still compiles to a working stack.
func applyComputeBindings(block *TFBlock, node GraphNode, b *edgeBindings, nb networkBindings) {
	subnetExpr := nb.subnetAExpr
	if subnets := b.subnetOf[node.ID]; len(subnets) > 0 {
		subnetExpr = "aws_subnet." + subnets[0] + ".id"
	}
	block.ExprAttributes["subnet_id"] = subnetExpr

	sgExpr := defaultEC2SecurityGroupExpr()
	if sgs := b.sgOf[node.ID]; len(sgs) > 0 {
		refs := make([]string, 0, len(sgs))
		for _, s := range sgs {
			refs = append(refs, "aws_security_group."+s+".id")
		}
		sgExpr = joinExprList(refs)
	}
	block.ExprAttributes["vpc_security_group_ids"] = sgExpr

	if block.Attributes["ami"] == "" {
		block.ExprAttributes["ami"] = "data.aws_ami." + napkinAMIData + ".id"
	}

	dbDeps := append([]string(nil), b.dbDependsOnEC2[node.ID]...)
	if !hasUserData(*block) {
		block.ExprAttributes["user_data"] = napkinEC2UserData(b.dbInjectIntoEC2[node.ID])
	}
	if profile := b.instanceProfileFor[node.ID]; profile != "" {
		block.ExprAttributes["iam_instance_profile"] = "aws_iam_instance_profile." + profile + "_profile.name"
	}

	if len(dbDeps) > 0 {
		refs := make([]string, 0, len(dbDeps))
		for _, slug := range dbDeps {
			refs = append(refs, "aws_db_instance."+slug)
		}
		block.ExprAttributes["depends_on"] = joinExprList(refs)
	}
}

// applyDatabaseBindings wires subnet groups + reverse depends_on for compute->db edges.
func applyDatabaseBindings(block *TFBlock, node GraphNode, b *edgeBindings) {
	if subnets := b.subnetOf[node.ID]; len(subnets) > 0 {
		block.ExprAttributes["db_subnet_group_name"] = "aws_db_subnet_group." + node.LocalName + "_sg.name"
	}
	if reverse := b.dbDependsOnEC2Reverse[node.ID]; len(reverse) > 0 {
		refs := make([]string, 0, len(reverse))
		for _, slug := range reverse {
			refs = append(refs, "aws_instance."+slug)
		}
		block.ExprAttributes["depends_on"] = joinExprList(refs)
	}
}

// applyLoadBalancerBindings wires subnets + security_groups on the LB.
// Falls back to napkin's two subnets / ALB SG when no edges are drawn.
func applyLoadBalancerBindings(block *TFBlock, node GraphNode, b *edgeBindings, nb networkBindings) {
	subnetExprs := make([]string, 0, 2)
	if subnets := b.subnetOf[node.ID]; len(subnets) > 0 {
		for _, sn := range subnets {
			subnetExprs = append(subnetExprs, "aws_subnet."+sn+".id")
		}
		// ALBs need >= 2 subnets across AZs; pad with napkin subnet B as a sane fallback.
		if len(subnetExprs) == 1 {
			subnetExprs = append(subnetExprs, nb.subnetBExpr)
		}
	} else {
		subnetExprs = []string{nb.subnetAExpr, nb.subnetBExpr}
	}
	block.ExprAttributes["subnets"] = "[" + strings.Join(subnetExprs, ", ") + "]"

	sgExpr := defaultLBSecurityGroupExpr()
	if sgs := b.sgOf[node.ID]; len(sgs) > 0 {
		refs := make([]string, 0, len(sgs))
		for _, s := range sgs {
			refs = append(refs, "aws_security_group."+s+".id")
		}
		sgExpr = joinExprList(refs)
	}
	block.ExprAttributes["security_groups"] = sgExpr
	block.ExprAttributes["internal"] = "false"
}

// applyLambdaBindings wires the iam role expr; event source mappings are
// emitted as separate top-level blocks below.
func applyLambdaBindings(block *TFBlock, node GraphNode, b *edgeBindings) {
	if role := b.lambdaRole[node.ID]; role != "" {
		block.ExprAttributes["role"] = "aws_iam_role." + role + ".arn"
	}
}

// emitDBSubnetGroups creates one aws_db_subnet_group per database that has
// subnet edges, since RDS requires this resource for VPC-bound databases.
func emitDBSubnetGroups(ir IR, b *edgeBindings) []TFBlock {
	var blocks []TFBlock
	for _, n := range ir.Nodes {
		if n.Type != "aws_db_instance" {
			continue
		}
		subnets := b.subnetOf[n.ID]
		if len(subnets) == 0 {
			continue
		}
		refs := make([]string, 0, len(subnets))
		for _, sn := range subnets {
			refs = append(refs, "aws_subnet."+sn+".id")
		}
		blocks = append(blocks, TFBlock{
			Class:  "resource",
			Labels: []string{"aws_db_subnet_group", n.LocalName + "_sg"},
			ExprAttributes: map[string]string{
				"subnet_ids": joinExprList(refs),
			},
		})
	}
	return blocks
}

// emitInstanceProfiles wraps each IAM role bound to an EC2 in an aws_iam_instance_profile.
func emitInstanceProfiles(ir IR, b *edgeBindings) []TFBlock {
	var blocks []TFBlock
	seen := map[string]bool{}
	for _, n := range ir.Nodes {
		if n.Type != "aws_instance" {
			continue
		}
		role := b.instanceProfileFor[n.ID]
		if role == "" || seen[role] {
			continue
		}
		seen[role] = true
		blocks = append(blocks, TFBlock{
			Class:  "resource",
			Labels: []string{"aws_iam_instance_profile", role + "_profile"},
			ExprAttributes: map[string]string{
				"role": "aws_iam_role." + role + ".name",
			},
		})
	}
	return blocks
}

// emitLambdaEventSources turns SQS->Lambda edges into aws_lambda_event_source_mapping.
func emitLambdaEventSources(ir IR, b *edgeBindings) []TFBlock {
	var blocks []TFBlock
	for _, n := range ir.Nodes {
		if n.Type != "aws_lambda_function" {
			continue
		}
		queues := b.queueToLambda[n.ID]
		if len(queues) == 0 {
			continue
		}
		for _, q := range queues {
			blocks = append(blocks, TFBlock{
				Class:  "resource",
				Labels: []string{"aws_lambda_event_source_mapping", n.LocalName + "_" + q + "_evt"},
				ExprAttributes: map[string]string{
					"event_source_arn": "aws_sqs_queue." + q + ".arn",
					"function_name":    "aws_lambda_function." + n.LocalName + ".function_name",
				},
			})
		}
	}
	return blocks
}

func defaultEC2SecurityGroupExpr() string {
	return "[aws_security_group." + napkinSGEC2 + ".id]"
}

func defaultLBSecurityGroupExpr() string {
	return "[aws_security_group." + napkinSGALB + ".id]"
}

func lbVPCExpr(lb GraphNode, b *edgeBindings, nb networkBindings) string {
	if subnets := b.subnetOf[lb.ID]; len(subnets) > 0 {
		// LB target groups need a vpc_id; use the VPC of the (first) explicit subnet
		// when we can't easily resolve it. Fall back to the canvas/napkin VPC.
		_ = subnets
	}
	return nb.vpcIDExpr
}

func hasVPCID(block TFBlock) bool {
	if block.Attributes["vpc_id"] != "" {
		return true
	}
	if block.ExprAttributes != nil {
		if v, ok := block.ExprAttributes["vpc_id"]; ok && v != "" {
			return true
		}
	}
	return false
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

func napkinSecurityGroupBlocks(vpcIDExpr string) []TFBlock {
	vpcID := vpcIDExpr
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

func napkinEC2UserData(rdsSlugs []string) string {
	var b strings.Builder
	b.WriteString("<<-EOT\n#!/bin/bash\nset -euo pipefail\n")
	b.WriteString("# Napkin Phase 1: serve HTTP on :80 for ALB health checks\n")
	b.WriteString("if command -v dnf >/dev/null 2>&1; then dnf install -y nginx; ")
	b.WriteString("elif command -v yum >/dev/null 2>&1; then yum install -y nginx; ")
	b.WriteString("else apt-get update && apt-get install -y nginx; fi\n")
	b.WriteString("systemctl enable nginx\nsystemctl start nginx\n")

	sortedSlugs := append([]string(nil), rdsSlugs...)
	slices.Sort(sortedSlugs)

	for i, slug := range sortedSlugs {
		ref := fmt.Sprintf("aws_db_instance.%s", slug)
		if len(sortedSlugs) == 1 {
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

func napkinLBTargetGroupBlock(tfName, namePrefix, vpcIDExpr string) TFBlock {
	return TFBlock{
		Class:      "resource",
		Labels:     []string{"aws_lb_target_group", tfName},
		Attributes: map[string]string{"name_prefix": namePrefix, "protocol": "HTTP"},
		ExprAttributes: map[string]string{
			"port":                 "80",
			"vpc_id":               vpcIDExpr,
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

func (t *TerraformTarget) ToString() string {
	return ""
}

var _ Target = (*TerraformTarget)(nil)
