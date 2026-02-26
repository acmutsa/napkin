package compiler

type TerraformTarget struct {
}

func (t *TerraformTarget) Compile(ir IR) (*TFFile, error) {
	tfFile := &TFFile{}

	for _, node := range ir.Nodes {
		attributes := make(map[string]string, len(node.Attributes))
		for field, value := range node.Attributes {
			attributes[field] = value
		}
		resource := TFResource{
			Type:       node.Type,
			Name:       node.ID,
			Attributes: attributes,
		}
		tfFile.Resources = append(tfFile.Resources, resource)
	}
	return tfFile, nil
}

var _ Target = (*TerraformTarget)(nil)
