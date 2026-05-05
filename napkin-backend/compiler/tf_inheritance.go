package compiler

import (
	"fmt"
	"slices"
	"strings"
)

// applyInheritance fills in two narrow classes of implicit bindings on top of
// the explicit edge resolution:
//
//  1. VPC inheritance for unwired subnets and security groups when there is
//     exactly one VPC on the canvas. Multiple canvas VPCs with at least one
//     unwired subnet/SG is rejected with a clear error so users don't get
//     surprised by an alphabetical pick.
//  2. Subnet inheritance for load balancers that have no explicit subnet edge
//     but do forward to compute targets which themselves have explicit subnet
//     edges. The LB inherits the union of those target subnets, padded with a
//     same-VPC canvas subnet when the union is a single subnet (ALBs require
//     two subnets in different AZs).
//
// Mutations are performed on b in-place. The returned map records, per node
// ID and field, the source(s) the binding was inherited from so the API/UI
// layer can surface "inherited from X" hints. A nil-or-empty return is fine
// when no inference fired.
func applyInheritance(ir IR, b *edgeBindings, nb networkBindings) (map[string]map[string]string, error) {
	inheritedFrom := map[string]map[string]string{}

	if err := inheritVPCBindings(ir, b, nb, inheritedFrom); err != nil {
		return nil, err
	}

	inheritLBSubnets(ir, b, nb, inheritedFrom)

	if len(inheritedFrom) == 0 {
		return nil, nil
	}
	return inheritedFrom, nil
}

func inheritVPCBindings(ir IR, b *edgeBindings, nb networkBindings, out map[string]map[string]string) error {
	for _, n := range ir.Nodes {
		if n.Class != ClassResource {
			continue
		}
		if n.Type != "aws_subnet" && n.Type != "aws_security_group" {
			continue
		}
		if existing, ok := b.vpcOf[n.ID]; ok && existing != "" {
			continue
		}
		switch len(nb.canvasVPCs) {
		case 0:
			// No canvas VPC; SG falls back to napkin's VPC in the applier.
			// (A subnet without a canvas VPC isn't reachable: subnets without
			// any VPC at all aren't valid input, and useCanvasNet=false implies
			// no canvas subnets either.)
		case 1:
			vpc := nb.canvasVPCs[0]
			b.vpcOf[n.ID] = vpc
			recordInheritance(out, n.ID, "vpc_id", vpc)
		default:
			return fmt.Errorf(
				"%s %q: cannot infer vpc_id with %d canvas VPCs (%s); draw an explicit vpc.network -> %s.vpc edge",
				n.Type, n.LocalName, len(nb.canvasVPCs), strings.Join(nb.canvasVPCs, ", "), vpcEdgeTargetPort(n.Type),
			)
		}
	}
	return nil
}

func vpcEdgeTargetPort(tfType string) string {
	if tfType == "aws_security_group" {
		return "securityGroup"
	}
	return "subnet"
}

func inheritLBSubnets(ir IR, b *edgeBindings, _ networkBindings, out map[string]map[string]string) {
	for _, n := range ir.Nodes {
		if n.Type != "aws_lb" {
			continue
		}
		if len(b.subnetOf[n.ID]) > 0 {
			continue
		}
		targets := b.lbTargets[n.ID]
		if len(targets) == 0 {
			continue
		}

		var inherited []string
		var sources []string
		for _, t := range targets {
			tSubnets := b.subnetOf[t.ID]
			if len(tSubnets) == 0 {
				continue
			}
			added := false
			for _, sn := range tSubnets {
				if !slices.Contains(inherited, sn) {
					inherited = append(inherited, sn)
					added = true
				}
			}
			if added {
				sources = appendUnique(sources, t.LocalName)
			}
		}
		if len(inherited) == 0 {
			continue
		}

		// ALB requires >= 2 subnets across different AZs. When inheritance only
		// found one, prefer another canvas subnet in the SAME VPC over crossing
		// into napkin's default network (which would be a different VPC and
		// fail at apply time). If no same-VPC partner exists, leave the list at
		// 1 and let applyLoadBalancerBindings pad with the napkin default.
		if len(inherited) == 1 {
			if pad := pickSameVPCSubnet(ir, b, inherited[0]); pad != "" {
				inherited = append(inherited, pad)
			}
		}

		b.subnetOf[n.ID] = inherited
		recordInheritance(out, n.ID, "subnets", strings.Join(sources, ", "))
	}
}

// pickSameVPCSubnet returns the LocalName of any canvas subnet that lives in
// the same VPC as exclude (other than exclude itself). Returns "" when no
// suitable partner is found.
func pickSameVPCSubnet(ir IR, b *edgeBindings, exclude string) string {
	var excludeVPC string
	for _, n := range ir.Nodes {
		if n.Type == "aws_subnet" && n.LocalName == exclude {
			excludeVPC = b.vpcOf[n.ID]
			break
		}
	}
	if excludeVPC == "" {
		return ""
	}
	candidates := make([]string, 0)
	for _, n := range ir.Nodes {
		if n.Type != "aws_subnet" || n.LocalName == exclude {
			continue
		}
		if b.vpcOf[n.ID] == excludeVPC {
			candidates = append(candidates, n.LocalName)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	slices.Sort(candidates)
	return candidates[0]
}

func recordInheritance(out map[string]map[string]string, nodeID, field, source string) {
	if out[nodeID] == nil {
		out[nodeID] = map[string]string{}
	}
	out[nodeID][field] = source
}
