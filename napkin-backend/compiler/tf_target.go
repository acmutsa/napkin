package compiler

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

type TerraformTarget struct {
}

func (t *TerraformTarget) Compile(ir IR) (*TFFile, error) {
	tfFile := &TFFile{}

	terraformBlock := TFBlock{
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

	providerBlock := TFBlock{
		Class:      "provider",
		Labels:     []string{"aws"},
		Attributes: map[string]string{"region": "us-east-1"},
	}

	tfFile.Block = append(tfFile.Block, terraformBlock, providerBlock)

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
			if slugs := rdsRefsByEC2[node.ID]; len(slugs) > 0 && !hasUserData(block) {
				slices.Sort(slugs)
				block.ExprAttributes["user_data"] = rdsEnvUserDataHeredoc(slugs)
			}
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

	return tfFile, nil
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

// rdsEnvUserDataHeredoc generates a cloud-init script that exports DB connection
// details from each linked aws_db_instance into /etc/environment on first boot.
// Terraform interpolates the ${...} references at apply time, so the script
// that lands on the EC2 instance has concrete hostnames and ports.
func rdsEnvUserDataHeredoc(dbLocalNames []string) string {
	var b strings.Builder
	b.WriteString("<<-EOT\n")
	b.WriteString("#!/bin/bash\n")
	b.WriteString("# Napkin: wire RDS into instance env (adjust paths/tools as needed)\n")
	for i, slug := range dbLocalNames {
		ref := fmt.Sprintf("aws_db_instance.%s", slug)
		if len(dbLocalNames) == 1 {
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
func edgeAllowsRDSLink(e DirectedEdge, from, to GraphNode) bool {
	switch {
	case from.Type == "aws_db_instance" && to.Type == "aws_instance":
		if e.TargetPort != "" && e.TargetPort != "db-in" {
			return false
		}
		if e.SourcePort != "" && e.SourcePort != "db-out" {
			return false
		}
		return true
	case from.Type == "aws_instance" && to.Type == "aws_db_instance":
		if e.SourcePort != "" && e.SourcePort != "data-out" && e.SourcePort != "network-out" {
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
