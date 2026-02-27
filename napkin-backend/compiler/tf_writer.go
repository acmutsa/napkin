package compiler

import (
	"os"
	"strconv"
	"strings"
)

func writeBlock(builder *strings.Builder, block TFBlock, indent string) {
	builder.WriteString(indent + block.Class)

	for _, label := range block.Labels {
		builder.WriteString(" " + strconv.Quote(label))
	}

	builder.WriteString(" {\n")

	for k, v := range block.Attributes {
		builder.WriteString(indent + "  " + k + " = " + strconv.Quote(v) + "\n")
	}

	for _, child := range block.Blocks {
		writeBlock(builder, child, indent+"  ")
	}

	builder.WriteString(indent + "}\n")
}

func WriteTerraformFile(tf *TFFile, path string) error {
	var builder strings.Builder

	for _, block := range tf.Block {
		writeBlock(&builder, block, "")
		builder.WriteString("\n")
	}

	return os.WriteFile(path, []byte(builder.String()), 0644)
}
