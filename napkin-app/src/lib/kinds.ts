/**
 * Frontend node-kind registry. Mirrors napkin-backend/graph/kinds.go.
 * Keep both in sync when adding a new kind, port type, or attribute default.
 *
 * Port model (V1):
 *  - Every port has an `id` (unique within its direction on a kind),
 *    a `type` from the closed PortType set, and a human label.
 *  - Edges MUST connect a source output port to a target input port whose
 *    `type` matches. The compiler dispatches wiring based on
 *    (sourceKind, sourcePortId) -> (targetKind, targetPortId), so port ids
 *    carry semantic meaning (e.g. "subnet", "instanceRole") rather than just
 *    a direction prefix.
 *
 * The backend Normalize() step is the source of truth for default attribute
 * values; this registry only needs entries that drive the canvas (label,
 * description, colors, ports, terraform type).
 */

export type PortType = "network" | "data" | "env" | "iam";

export interface PortDef {
  id: string;
  type: PortType;
  label: string;
}

export interface KindDef {
  /** Stable id, matches Go NodeKind. Sent on the wire as `node.kind`. */
  kind: string;
  /** Human-friendly title shown on the node and in the toolbox. */
  label: string;
  /** Toolbox sub-text. */
  description: string;
  /** Terraform resource type used by the compile target. */
  terraformType: string;
  color: string;
  borderColor: string;
  iconColor: string;
  inputs: PortDef[];
  outputs: PortDef[];
}

/** Tailwind classes applied to the handle dot, keyed by port type. */
export const PORT_TYPE_HANDLE_CLASS: Record<PortType, string> = {
  network: "!bg-blue-400 !border-blue-500",
  data: "!bg-gray-400 !border-gray-500",
  env: "!bg-green-400 !border-green-500",
  iam: "!bg-purple-400 !border-purple-500",
};

export const KINDS: Record<string, KindDef> = {
  vpc: {
    kind: "vpc",
    label: "VPC",
    description: "Virtual network",
    terraformType: "aws_vpc",
    color: "bg-slate-50 border-slate-200",
    borderColor: "border-slate-200",
    iconColor: "text-slate-400",
    inputs: [],
    outputs: [{ id: "network", type: "network", label: "Network" }],
  },
  subnet: {
    kind: "subnet",
    label: "Subnet",
    description: "VPC subnet",
    terraformType: "aws_subnet",
    color: "bg-cyan-50 border-cyan-200",
    borderColor: "border-cyan-200",
    iconColor: "text-cyan-400",
    inputs: [{ id: "vpc", type: "network", label: "Parent VPC" }],
    outputs: [{ id: "placement", type: "network", label: "Placement" }],
  },
  securityGroup: {
    kind: "securityGroup",
    label: "Security Group",
    description: "Firewall rules",
    terraformType: "aws_security_group",
    color: "bg-yellow-50 border-yellow-200",
    borderColor: "border-yellow-200",
    iconColor: "text-yellow-400",
    inputs: [{ id: "vpc", type: "network", label: "VPC" }],
    outputs: [{ id: "attachment", type: "network", label: "Attach to" }],
  },
  iamRole: {
    kind: "iamRole",
    label: "IAM Role",
    description: "Identity & permissions",
    terraformType: "aws_iam_role",
    color: "bg-rose-50 border-rose-200",
    borderColor: "border-rose-200",
    iconColor: "text-rose-400",
    inputs: [],
    outputs: [{ id: "role", type: "iam", label: "Role" }],
  },
  compute: {
    kind: "compute",
    label: "EC2 Instance",
    description: "Virtual server",
    terraformType: "aws_instance",
    color: "bg-blue-50 border-blue-200",
    borderColor: "border-blue-200",
    iconColor: "text-blue-400",
    inputs: [
      { id: "subnet", type: "network", label: "Subnet" },
      { id: "securityGroup", type: "network", label: "Security Group" },
      { id: "inboundTraffic", type: "network", label: "Inbound Traffic" },
      { id: "instanceRole", type: "iam", label: "Instance Role" },
    ],
    outputs: [
      { id: "outboundTraffic", type: "network", label: "Outbound Traffic" },
      { id: "dataSource", type: "data", label: "Data Source" },
    ],
  },
  database: {
    kind: "database",
    label: "Database",
    description: "Managed RDS instance",
    terraformType: "aws_db_instance",
    color: "bg-green-50 border-green-200",
    borderColor: "border-green-200",
    iconColor: "text-green-400",
    inputs: [
      { id: "subnet", type: "network", label: "Subnet" },
      { id: "inboundTraffic", type: "network", label: "Inbound Traffic" },
      { id: "connection", type: "data", label: "DB Connection" },
    ],
    outputs: [],
  },
  loadBalancer: {
    kind: "loadBalancer",
    label: "Load Balancer",
    description: "Distributes traffic",
    terraformType: "aws_lb",
    color: "bg-purple-50 border-purple-200",
    borderColor: "border-purple-200",
    iconColor: "text-purple-400",
    inputs: [
      { id: "subnet", type: "network", label: "Subnet" },
      { id: "securityGroup", type: "network", label: "Security Group" },
      { id: "inboundTraffic", type: "network", label: "Public Traffic" },
    ],
    outputs: [
      { id: "forward", type: "network", label: "Forward to Targets" },
    ],
  },
  lambda: {
    kind: "lambda",
    label: "Lambda",
    description: "Serverless function",
    terraformType: "aws_lambda_function",
    color: "bg-amber-50 border-amber-200",
    borderColor: "border-amber-200",
    iconColor: "text-amber-400",
    inputs: [
      { id: "executionRole", type: "iam", label: "Execution Role" },
      { id: "eventSource", type: "data", label: "Event Source" },
    ],
    outputs: [{ id: "output", type: "data", label: "Output" }],
  },
  sqsQueue: {
    kind: "sqsQueue",
    label: "SQS Queue",
    description: "Message queue",
    terraformType: "aws_sqs_queue",
    color: "bg-pink-50 border-pink-200",
    borderColor: "border-pink-200",
    iconColor: "text-pink-400",
    inputs: [{ id: "producer", type: "data", label: "Producer" }],
    outputs: [{ id: "consumer", type: "data", label: "Consumer" }],
  },
  storageBucket: {
    kind: "storageBucket",
    label: "Storage Bucket",
    description: "Object storage",
    terraformType: "aws_s3_bucket",
    color: "bg-orange-50 border-orange-200",
    borderColor: "border-orange-200",
    iconColor: "text-orange-400",
    inputs: [
      { id: "producer", type: "data", label: "Objects In" },
      { id: "accessPolicy", type: "iam", label: "Access Policy" },
    ],
    outputs: [{ id: "consumer", type: "data", label: "Objects Out" }],
  },
};

/** Ordered list for stable toolbox rendering. */
export const KIND_LIST: KindDef[] = [
  KINDS.vpc,
  KINDS.subnet,
  KINDS.securityGroup,
  KINDS.iamRole,
  KINDS.compute,
  KINDS.database,
  KINDS.loadBalancer,
  KINDS.lambda,
  KINDS.sqsQueue,
  KINDS.storageBucket,
];

export function getKind(kind: string | undefined): KindDef | undefined {
  if (!kind) return undefined;
  return KINDS[kind];
}

/** Lookup helpers for connection validation. */
export function getOutputPort(
  kind: string | undefined,
  portId: string | undefined,
): PortDef | undefined {
  const def = getKind(kind);
  if (!def || !portId) return undefined;
  return def.outputs.find((p) => p.id === portId);
}

export function getInputPort(
  kind: string | undefined,
  portId: string | undefined,
): PortDef | undefined {
  const def = getKind(kind);
  if (!def || !portId) return undefined;
  return def.inputs.find((p) => p.id === portId);
}

/**
 * Returns null if a connection between (sourceKind/sourcePort) ->
 * (targetKind/targetPort) is allowed; otherwise returns a human-friendly
 * reason. Both ports must be specified, and both port types must match.
 */
export function validateConnection(args: {
  sourceKind: string | undefined;
  sourcePortId: string | undefined | null;
  targetKind: string | undefined;
  targetPortId: string | undefined | null;
}): string | null {
  const { sourceKind, sourcePortId, targetKind, targetPortId } = args;
  if (!sourcePortId || !targetPortId) {
    return "Connection requires both a source and target port";
  }
  const sourcePort = getOutputPort(sourceKind, sourcePortId);
  if (!sourcePort) {
    return `Unknown output port "${sourcePortId}" on ${sourceKind ?? "<unknown>"}`;
  }
  const targetPort = getInputPort(targetKind, targetPortId);
  if (!targetPort) {
    return `Unknown input port "${targetPortId}" on ${targetKind ?? "<unknown>"}`;
  }
  if (sourcePort.type !== targetPort.type) {
    return `Incompatible port types: ${sourceKind}.${sourcePortId} (${sourcePort.type}) cannot connect to ${targetKind}.${targetPortId} (${targetPort.type})`;
  }
  return null;
}
