import { useReactFlow } from '@xyflow/react';
import type { Node } from '@xyflow/react';
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

type Attributes = Record<string, string>;

type ResourceNodeData = {
  spec: NodeSpec;
  attributes: Attributes;
};

type ResourceNodeProps = {
  id: string;
  data: ResourceNodeData;
};

export default function ResourceNode({ id, data }: ResourceNodeProps) {
  const { spec, attributes } = data;
  const { deleteElements, setNodes } = useReactFlow<Node<ResourceNodeData>>();

  const handleDelete = () => {
    deleteElements({ nodes: [{ id }] });
  };

  const addAttribute = () => {
    const newKey = `key-${Date.now()}`;
    setNodes((nodes) =>
      nodes.map((node) =>
        node.id === id
          ? {
              ...node,
              data: {
                ...node.data,
                attributes: { ...node.data.attributes, [newKey]: "" },
              },
            }
          : node
      )
    );
  };

  const updateAttribute = (key: string, value: string) => {
    setNodes((nodes) =>
      nodes.map((node) =>
        node.id === id
          ? {
              ...node,
              data: {
                ...node.data,
                attributes: { ...node.data.attributes, [key]: value },
              },
            }
          : node
      )
    );
  };

  const renameAttribute = (oldKey: string, newKey: string) => {
    if (!newKey || oldKey === newKey) return;
    setNodes((nodes) =>
      nodes.map((node) => {
        if (node.id !== id) return node;
        const currentAttrs = node.data.attributes;
        if (currentAttrs[newKey]) return node;
        const value = currentAttrs[oldKey];
        const updatedAttrs: Attributes = { ...currentAttrs };
        delete updatedAttrs[oldKey];
        updatedAttrs[newKey] = value;
        return { ...node, data: { ...node.data, attributes: updatedAttrs } };
      })
    );
  };

  const deleteAttribute = (key: string) => {
    setNodes((nodes) =>
      nodes.map((node) => {
        if (node.id !== id) return node;
        const updatedAttrs = { ...node.data.attributes };
        delete updatedAttrs[key];
        return { ...node, data: { ...node.data, attributes: updatedAttrs } };
      })
    );
  };

  return (
    <BaseNode className={spec.color}>
      <BaseNodeHeader className={`border-b ${spec.borderColor}`}>
        <BaseNodeHeaderTitle>{spec.label}</BaseNodeHeaderTitle>
        <Button
          variant="ghost"
          className="nodrag p-1 !bg-transparent hover:bg-gray-200/20"
          onClick={handleDelete}
        >
          <Trash className={`size-4 ${spec.iconColor}`} />
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

        <div className="mt-2 space-y-1 px-2">
          {Object.entries(attributes).map(([key, value]) => (
            <div key={key} className="flex gap-1 items-center">
              <input
                placeholder="Attribute name"
                defaultValue={key}
                onBlur={(e) => renameAttribute(key, e.target.value)}
                className={`w-1/2 px-1 py-0.5 text-xs h-6 rounded border 
                  ${spec.borderColor} bg-white/60 dark:bg-black/30`}
              />
              <input
                value={value}
                onChange={(e) => updateAttribute(key, e.target.value)}
                className={`w-1/2 px-1 py-0.5 text-xs h-6 rounded border 
                  ${spec.borderColor}`}
              />
              <Button
                variant="ghost"
                size="icon"
                className="p-0.5 !bg-transparent hover:bg-gray-200/20"
                onClick={() => deleteAttribute(key)}
                title="Delete attribute"
              >
                <Trash className={`size-3 ${spec.iconColor}`} />
              </Button>
            </div>
          ))}

          <Button
            variant="ghost"
            size="sm"
            className="w-full text-xs py-1 mt-1 !bg-transparent hover:bg-gray-200/20"
            onClick={addAttribute}
          >
            + Add Attribute
          </Button>
        </div>
      </BaseNodeContent>
    </BaseNode>
  );
}