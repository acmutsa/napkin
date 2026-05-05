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
import { Badge } from "@/components/ui/badge";

import { Trash } from 'lucide-react';
import { Position } from '@xyflow/react';

import {
  PORT_TYPE_HANDLE_CLASS,
  type KindDef,
  type PortDef,
} from '@/lib/kinds';

type Attributes = Record<string, string>;

type ResourceNodeData = {
  kind?: string;
  spec: KindDef;
  attributes: Attributes;
  error?: string | null;
  /** Map of HCL field name to source string for compiler-inherited bindings. */
  inherited?: Record<string, string> | null;
};

type ResourceNodeProps = {
  id: string;
  data: ResourceNodeData;
};

function handleClass(port: PortDef): string {
  return PORT_TYPE_HANDLE_CLASS[port.type] ?? "!bg-gray-400 !border-gray-500";
}

export default function ResourceNode({ id, data }: ResourceNodeProps) {
  const { spec, attributes, error, inherited } = data;
  const { deleteElements, setNodes } = useReactFlow<Node<ResourceNodeData>>();
  const inheritedEntries = inherited ? Object.entries(inherited) : [];

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
        if (newKey in currentAttrs) return node;
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
          aria-label="Delete Node"
          title="Delete Node"
        >
          <Trash className={`size-4 ${spec.iconColor}`} />
        </Button>
      </BaseNodeHeader>

      <BaseNodeContent className="text-xs space-y-1 px-0 py-2">
        {error && (
          <div className="mx-2 mb-2 rounded-md border border-red-300 bg-red-50 px-2 py-1 text-[11px] text-red-700">
            {error}
          </div>          
        )}

        {inheritedEntries.length > 0 && (
          <div className="mx-2 mb-2 flex flex-wrap gap-1">
            {inheritedEntries.map(([field, source]) => (
              <Badge
                key={field}
                variant="secondary"
                className="text-[10px] font-normal"
                title={`${field} inherited from ${source}`}
              >
                {field} from {source}
              </Badge>
            ))}
          </div>
        )}

        {spec.inputs?.map((port) => (
          <LabeledHandle
            key={port.id}
            id={port.id}
            type="target"
            position={Position.Left}
            title={port.label}
            handleClassName={handleClass(port)}
          />
        ))}

        {spec.outputs?.map((port) => (
          <LabeledHandle
            key={port.id}
            id={port.id}
            type="source"
            position={Position.Right}
            title={port.label}
            handleClassName={handleClass(port)}
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