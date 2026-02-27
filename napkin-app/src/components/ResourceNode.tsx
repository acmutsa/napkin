import { Handle, Position } from '@xyflow/react';
import { Card, CardHeader, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

type PortType = "network" | "data";

type PortSpec = {
  id: string;
  type: PortType;
  label: string;
};

type NodeSpec = {
  label: string;
  color?: string;
  inputs?: PortSpec[];
  outputs?: PortSpec[];
};

type ResourceNodeProps = {
  data: {
    spec: NodeSpec;
  };
};

export default function ResourceNode({ data }: ResourceNodeProps) {
  const { spec } = data;
  const isEC2 = spec.label === "EC2 Instance";

  return (
    <Card className={`relative w-56 shadow-md ${spec.color}`}>
      <CardHeader className="text-sm font-semibold">{spec.label}</CardHeader>
      <CardContent className="relative text-xs space-y-2">

        {spec.inputs?.map((port, index) => (
          <div key={port.id} className="relative flex items-center" style={{ height: '40px' }}>
            <Handle
              type="target"
              position={Position.Left}
              id={port.id}
              style={{ top: isEC2 ? 35 : 40 + index * 20 }}
            />
            <Badge variant="outline" className="ml-2">
              {port.label}
            </Badge>
          </div>
        ))}

        {spec.outputs?.map((port, index) => (
          <div key={port.id} className="relative flex justify-end items-center" style={{ height: '40px' }}>
            <Badge variant="outline" className="mr-2">
              {port.label}
            </Badge>
            <Handle
              type="source"
              position={Position.Right}
              id={port.id}
              style={{ top: isEC2 ? 35 : 40 + index * 20 }}
            />
          </div>
        ))}
      </CardContent>
    </Card>
  );
}