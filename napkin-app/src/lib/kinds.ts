/**
 * Frontend node-kind registry. Mirrors napkin-backend/graph/kinds.go.
 * Keep both in sync when adding a new kind, port type, or attribute default.
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
  compute: {
    kind: "compute",
    label: "EC2 Instance",
    description: "Virtual server",
    terraformType: "aws_instance",
    color: "bg-blue-50 border-blue-200",
    borderColor: "border-blue-200",
    iconColor: "text-blue-400",
    inputs: [
      { id: "in-network", type: "network", label: "Incoming Traffic" },
      { id: "in-env", type: "env", label: "Environment" },
      { id: "in-iam", type: "iam", label: "IAM Role" },
    ],
    outputs: [
      { id: "out-network", type: "network", label: "Outbound Traffic" },
      { id: "out-data", type: "data", label: "Data Out" },
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
      { id: "in-network", type: "network", label: "Network" },
      { id: "in-env", type: "env", label: "Configuration" },
    ],
    outputs: [{ id: "out-data", type: "data", label: "Connection" }],
  },
  loadBalancer: {
    kind: "loadBalancer",
    label: "Load Balancer",
    description: "Distributes traffic",
    terraformType: "aws_lb",
    color: "bg-purple-50 border-purple-200",
    borderColor: "border-purple-200",
    iconColor: "text-purple-400",
    inputs: [{ id: "in-network", type: "network", label: "Incoming Traffic" }],
    outputs: [{ id: "out-network", type: "network", label: "Forward Traffic" }],
  },
  securityGroup: {
    kind: "securityGroup",
    label: "Security Group",
    description: "Firewall rules",
    terraformType: "aws_security_group",
    color: "bg-yellow-50 border-yellow-200",
    borderColor: "border-yellow-200",
    iconColor: "text-yellow-400",
    inputs: [{ id: "in-network", type: "network", label: "Inbound" }],
    outputs: [{ id: "out-network", type: "network", label: "Outbound" }],
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
      { id: "in-data", type: "data", label: "Objects In" },
      { id: "in-iam", type: "iam", label: "Access Policy" },
    ],
    outputs: [{ id: "out-data", type: "data", label: "Objects Out" }],
  },
  vpc: {
    kind: "vpc",
    label: "VPC",
    description: "Virtual network",
    terraformType: "aws_vpc",
    color: "bg-slate-50 border-slate-200",
    borderColor: "border-slate-200",
    iconColor: "text-slate-400",
    inputs: [],
    outputs: [{ id: "out-network", type: "network", label: "Network" }],
  },
  subnet: {
    kind: "subnet",
    label: "Subnet",
    description: "VPC subnet",
    terraformType: "aws_subnet",
    color: "bg-cyan-50 border-cyan-200",
    borderColor: "border-cyan-200",
    iconColor: "text-cyan-400",
    inputs: [{ id: "in-network", type: "network", label: "Parent VPC" }],
    outputs: [{ id: "out-network", type: "network", label: "Network" }],
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
    outputs: [{ id: "out-iam", type: "iam", label: "Role" }],
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
      { id: "in-iam", type: "iam", label: "Execution Role" },
      { id: "in-env", type: "env", label: "Environment" },
      { id: "in-data", type: "data", label: "Event Source" },
    ],
    outputs: [
      { id: "out-network", type: "network", label: "Outbound Calls" },
      { id: "out-data", type: "data", label: "Data Out" },
    ],
  },
  sqsQueue: {
    kind: "sqsQueue",
    label: "SQS Queue",
    description: "Message queue",
    terraformType: "aws_sqs_queue",
    color: "bg-pink-50 border-pink-200",
    borderColor: "border-pink-200",
    iconColor: "text-pink-400",
    inputs: [
      { id: "in-data", type: "data", label: "Producer" },
      { id: "in-iam", type: "iam", label: "Access Policy" },
    ],
    outputs: [{ id: "out-data", type: "data", label: "Consumer" }],
  },
};

/** Ordered list for stable toolbox rendering. */
export const KIND_LIST: KindDef[] = [
  KINDS.compute,
  KINDS.database,
  KINDS.loadBalancer,
  KINDS.securityGroup,
  KINDS.storageBucket,
  KINDS.vpc,
  KINDS.subnet,
  KINDS.iamRole,
  KINDS.lambda,
  KINDS.sqsQueue,
];

export function getKind(kind: string | undefined): KindDef | undefined {
  if (!kind) return undefined;
  return KINDS[kind];
}
