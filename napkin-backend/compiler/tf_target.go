package compiler

type TerraformTarget struct {
}

func (t *TerraformTarget) Compile(ir IR) (*TFFile, error) {
	tfFile := &TFFile{}

	for _, node := range ir.Nodes {
		resource := TFResource{
			Type:       node.Type,
			Name:       node.ID,
			Attributes: node.Attributes,
		}
		tfFile.Resources = append(tfFile.Resources, resource)
	}
	return tfFile, nil
}

var _ Target = (*TerraformTarget)(nil)
