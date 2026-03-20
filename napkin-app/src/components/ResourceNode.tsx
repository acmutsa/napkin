import { useReactFlow } from '@xyflow/react';
import { LabeledHandle } from '@/components/labeled-handle';
import {
  BaseNode,
  BaseNodeHeader,
  BaseNodeHeaderTitle,
  BaseNodeContent,
} from '@/components/base-node';
import { Button } from "@/components/ui/button";

import { Trash } from 'lucide-react';
import { Position } from '@xyflow/react';

type PortType = "network" | "data";

type PortSpec = {
  id: string;
  type: PortType;
  label: string;
};

type NodeSpec = {
  label: string;
  color?: string;
  borderColor?: string;
  iconColor?: string; 
  inputs?: PortSpec[];
  outputs?: PortSpec[];
};

type ResourceNodeProps = {
  id: string;
  data: {
    spec: NodeSpec;
  };
};

export default function ResourceNode({ id, data }: ResourceNodeProps) {
  const { spec } = data;
  const { deleteElements } = useReactFlow();

  function handleDelete() {
    deleteElements({ nodes: [{ id }] });
  }

  return (
    <BaseNode className={spec.color}>

      <BaseNodeHeader className={`border-b ${spec.borderColor}`}>
        <BaseNodeHeaderTitle>{spec.label}</BaseNodeHeaderTitle>
        <Button
          variant="ghost"
          className="nodrag p-1 !bg-transparent"
          onClick={handleDelete}
          aria-label="Delete Node"
          title="Delete Node"
        >
          <Trash className={`size-4 ${spec.iconColor}`}/>
        </Button>
      </BaseNodeHeader>

      <BaseNodeContent className="text-xs space-y-1 px-0 py-2">
        {spec.inputs?.map((port) => (
          <LabeledHandle
            key={port.id}
            id={port.id}
            type="target"
            position={Position.Left}
            title={port.label}
            handleClassName="!bg-yellow-400 !border-yellow-500"
          />
        ))}

        {spec.outputs?.map((port) => (
          <LabeledHandle
            key={port.id}
            id={port.id}
            type="source"
            position={Position.Right}
            title={port.label}
            handleClassName="!bg-gray-400 !border-gray-500"
          />
        ))}

      </BaseNodeContent>
    </BaseNode>
  );
}