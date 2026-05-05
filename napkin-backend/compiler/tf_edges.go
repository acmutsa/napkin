package compiler

import (
	"fmt"
	"slices"
	"strings"
)

// edgeKey identifies a wiring rule by the (kind, port id) pair on each side.
// The compile target rejects any edge that does not match a registered key.
type edgeKey struct {
	fromKind string
	fromPort string
	toKind   string
	toPort   string
}

func keyFor(e DirectedEdge, from, to GraphNode) edgeKey {
	return edgeKey{
		fromKind: from.Kind,
		fromPort: e.SourcePort,
		toKind:   to.Kind,
		toPort:   e.TargetPort,
	}
}

// edgeEffect is the resolved per-edge contribution to the Terraform output.
// Effects are accumulated in compileContext during edgeAnalysis and then
// applied while emitting node blocks (and as standalone blocks where needed).
type edgeEffect string

const (
	effectVPCToSubnet           edgeEffect = "vpc->subnet"
	effectVPCToSG               edgeEffect = "vpc->sg"
	effectSubnetToCompute       edgeEffect = "subnet->compute"
	effectSubnetToDatabase      edgeEffect = "subnet->database"
	effectSubnetToLoadBalancer  edgeEffect = "subnet->lb"
	effectSGToCompute           edgeEffect = "sg->compute"
	effectSGToLoadBalancer      edgeEffect = "sg->lb"
	effectLBToCompute           edgeEffect = "lb->compute"
	effectComputeToDB           edgeEffect = "compute->db"
	effectDBToCompute           edgeEffect = "db->compute"
	effectIAMToCompute          edgeEffect = "iam->compute"
	effectIAMToLambda           edgeEffect = "iam->lambda"
	effectQueueToLambda         edgeEffect = "queue->lambda"
)

// edgeRegistry enumerates every (sourceKind.port -> targetKind.port) tuple
// the compiler knows how to materialize. Edges that pass IntentGraph
// validation but are not present here cause compilation to fail loudly.
var edgeRegistry = map[edgeKey]edgeEffect{
	{"vpc", "network", "subnet", "vpc"}:                   effectVPCToSubnet,
	{"vpc", "network", "securityGroup", "vpc"}:            effectVPCToSG,
	{"subnet", "placement", "compute", "subnet"}:          effectSubnetToCompute,
	{"subnet", "placement", "database", "subnet"}:         effectSubnetToDatabase,
	{"subnet", "placement", "loadBalancer", "subnet"}:     effectSubnetToLoadBalancer,
	{"securityGroup", "attachment", "compute", "securityGroup"}:        effectSGToCompute,
	{"securityGroup", "attachment", "loadBalancer", "securityGroup"}:   effectSGToLoadBalancer,
	{"loadBalancer", "forward", "compute", "inboundTraffic"}:           effectLBToCompute,
	{"compute", "outboundTraffic", "database", "inboundTraffic"}:       effectComputeToDB,
	{"compute", "dataSource", "database", "connection"}:                effectDBToCompute,
	{"iamRole", "role", "compute", "instanceRole"}:        effectIAMToCompute,
	{"iamRole", "role", "lambda", "executionRole"}:        effectIAMToLambda,
	{"sqsQueue", "consumer", "lambda", "eventSource"}:     effectQueueToLambda,
}

// edgeBindings is the aggregated, per-target-node view of the wirings to apply.
// It is built once per Compile() call and consulted by both the node-emission
// loop and the post-emission "synthetic block" loop (target groups, event source
// mappings, etc.).
type edgeBindings struct {
	// vpcOf[nodeID] = aws_vpc local name to bind for subnet/SG vpc_id.
	vpcOf map[string]string

	// subnetOf[nodeID] = aws_subnet local names attached as placement (ordered).
	// EC2 takes the first; LB needs >= 2; DB aggregates into a subnet group.
	subnetOf map[string][]string

	// sgOf[nodeID] = aws_security_group local names attached.
	sgOf map[string][]string

	// lbTargets[lbID] = compute nodes routed to from this LB.
	lbTargets map[string][]GraphNode

	// dbDependsOnLB[ec2ID] = aws_db_instance local names the EC2 depends on.
	dbDependsOnEC2 map[string][]string

	// dbInjectIntoEC2[ec2ID] = aws_db_instance local names whose endpoint env
	// vars should be injected into the EC2's user_data.
	dbInjectIntoEC2 map[string][]string

	// roleOf[ec2ID] = aws_iam_role local name to wrap in an instance profile.
	instanceProfileFor map[string]string
	// lambdaRole[lambdaID] = aws_iam_role local name.
	lambdaRole map[string]string

	// queueToLambda[lambdaID] = aws_sqs_queue local names whose ARN drives a
	// lambda event source mapping.
	queueToLambda map[string][]string
}

func newEdgeBindings() *edgeBindings {
	return &edgeBindings{
		vpcOf:                 map[string]string{},
		subnetOf:              map[string][]string{},
		sgOf:                  map[string][]string{},
		lbTargets:             map[string][]GraphNode{},
		dbDependsOnEC2:     map[string][]string{},
		dbInjectIntoEC2:    map[string][]string{},
		instanceProfileFor: map[string]string{},
		lambdaRole:            map[string]string{},
		queueToLambda:         map[string][]string{},
	}
}

// resolveEdges walks every directed edge and routes it through edgeRegistry.
// Unknown (kind, port) tuples produce a hard error so the canvas-to-Terraform
// mapping stays exhaustive: every connection the user draws shows up in the
// generated config (or compilation fails with a helpful pointer).
func resolveEdges(edges []DirectedEdge, idToNode map[string]GraphNode) (*edgeBindings, error) {
	b := newEdgeBindings()

	for _, e := range edges {
		from, ok1 := idToNode[e.FromID]
		to, ok2 := idToNode[e.ToID]
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("edge %s -> %s: unknown node", e.FromID, e.ToID)
		}
		effect, ok := edgeRegistry[keyFor(e, from, to)]
		if !ok {
			return nil, fmt.Errorf(
				"unsupported connection %s.%s -> %s.%s: no compiler wiring registered",
				from.Kind, e.SourcePort, to.Kind, e.TargetPort,
			)
		}

		switch effect {
		case effectVPCToSubnet, effectVPCToSG:
			b.vpcOf[to.ID] = from.LocalName
		case effectSubnetToCompute, effectSubnetToDatabase, effectSubnetToLoadBalancer:
			b.subnetOf[to.ID] = appendUnique(b.subnetOf[to.ID], from.LocalName)
		case effectSGToCompute, effectSGToLoadBalancer:
			b.sgOf[to.ID] = appendUnique(b.sgOf[to.ID], from.LocalName)
		case effectLBToCompute:
			b.lbTargets[from.ID] = append(b.lbTargets[from.ID], to)
		case effectComputeToDB:
			// Network-path edge only; Terraform ordering is handled by implicit
			// references when dataSource->connection is also present (EC2 user_data).
		case effectDBToCompute:
			// compute.dataSource -> database.connection
			// Even though the edge direction is compute -> database (to match the canvas
			// left-to-right flow), the effect is still "DB injects connection details
			// into the compute node", so we bind the target DB to the source EC2.
			b.dbDependsOnEC2[from.ID] = appendUnique(b.dbDependsOnEC2[from.ID], to.LocalName)
			b.dbInjectIntoEC2[from.ID] = appendUnique(b.dbInjectIntoEC2[from.ID], to.LocalName)
		case effectIAMToCompute:
			b.instanceProfileFor[to.ID] = from.LocalName
		case effectIAMToLambda:
			b.lambdaRole[to.ID] = from.LocalName
		case effectQueueToLambda:
			b.queueToLambda[to.ID] = appendUnique(b.queueToLambda[to.ID], from.LocalName)
		default:
			return nil, fmt.Errorf("internal error: unhandled edge effect %q", effect)
		}
	}

	return b, nil
}

func appendUnique(list []string, s string) []string {
	if s == "" {
		return list
	}
	if slices.Contains(list, s) {
		return list
	}
	return append(list, s)
}

// joinExprList renders a deterministic Terraform list expression like
// "[a, b, c]" from a slice of HCL expressions.
func joinExprList(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	sorted := append([]string(nil), items...)
	slices.Sort(sorted)
	return "[" + strings.Join(sorted, ", ") + "]"
}
