package compiler

import (
	"os"
	"slices"
	"strconv"
	"strings"
)

func sortedAttrKeys(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func writeBlock(builder *strings.Builder, block TFBlock, indent string) {
	if block.RawLine != "" {
		builder.WriteString(indent + block.RawLine + "\n")
		return
	}

	builder.WriteString(indent + block.Class)

	for _, label := range block.Labels {
		builder.WriteString(" " + strconv.Quote(label))
	}

	builder.WriteString(" {\n")

	for _, k := range sortedAttrKeys(block.Attributes) {
		v := block.Attributes[k]
		builder.WriteString(indent + "  " + k + " = " + strconv.Quote(v) + "\n")
	}

	for _, k := range sortedAttrKeys(block.ExprAttributes) {
		v := block.ExprAttributes[k]
		builder.WriteString(indent + "  " + k + " = " + v + "\n")
	}

	for _, child := range block.Blocks {
		writeBlock(builder, child, indent+"  ")
	}

	builder.WriteString(indent + "}\n")
}

func (tf *TFFile) String() string {
	var builder strings.Builder

	for _, block := range tf.Block {
		writeBlock(&builder, block, "")
		builder.WriteString("\n")
	}

	return builder.String()
}

func WriteTerraformFile(tf *TFFile, path string) error {
	return os.WriteFile(path, []byte(tf.String()), 0644)
}
