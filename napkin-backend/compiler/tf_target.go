package compiler

import (
	"fmt"
	"maps"
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
	tfFile.Block = append(tfFile.Block, terraformBlock)
	for _, node := range ir.Nodes {
		fmt.Printf("ID=%s Type=%s Class=%s\n", node.ID, node.Type, node.Class)
		block := TFBlock{
			Class:      string(node.Class),
			Attributes: make(map[string]string),
		}

		maps.Copy(block.Attributes, node.Attributes)
		switch node.Class {
		case ClassResource:
			block.Labels = []string{node.Type, node.ID}

		case ClassData:
			block.Labels = []string{node.Type, node.ID}

		case ClassProvider:
			block.Labels = []string{node.Type}

		case ClassModule:
			block.Labels = []string{node.ID}

		default:
			return nil, fmt.Errorf("unsupported node class: %s", node.Class)
		}
		tfFile.Block = append(tfFile.Block, block)
	}
	return tfFile, nil
}

var _ Target = (*TerraformTarget)(nil)
