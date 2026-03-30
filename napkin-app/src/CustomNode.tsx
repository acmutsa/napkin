import { Handle, Position, useReactFlow } from '@xyflow/react';
import type { Node, NodeProps } from '@xyflow/react';
 
type NumberNode = Node<{ kind: 'number'; number: number }, 'number'>;
type TextNode = Node<{ kind: 'text'; text: string }, 'text'>;

type AppNode = NumberNode | TextNode;

export default function CustomNode({ id, data }: NodeProps<AppNode>) {
    const { deleteElements } = useReactFlow();

  const handleDelete = () => {
    deleteElements({ nodes: [{ id }] });
  };
  
  return (
    <div className='relative bg-white rounded-md px-4 py-3 min-w-40 border transition-all duration-200'>
        <Handle 
        type='target' 
        position={Position.Top} 
        className='!w-2.5 !h-2.5 !bg-yellow-400 !border-2 !border-white'
        />

        <button
        onClick={handleDelete}
        title="Delete node"
        className="absolute -bottom-2 right-0 !bg-transparent !border-none text-red-400"
        >
            x
        </button>

        {data.kind === 'number' ? (
            <div>
                <p className="flex justify-center text-2xl text-black">
                    {data.number}
                </p>
            </div>
        ) : (
            <div>
                <p className="flex justify-center text-2xl text-black">
                    {data.text}
                </p>
            </div>
        )}

        <Handle
        type="source"
        position={Position.Bottom}
        className="!w-2.5 !h-2.5 !bg-gray-400 !border-2 !border-white"
        />
    </div>
  )
}