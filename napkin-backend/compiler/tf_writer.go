package compiler

import (
	"os"
	"strconv"
	"strings"
)

func WriteTerraformFile(tf *TFFile, path string) error {
	var builder strings.Builder

	for _, v := range tf.Resources {
		builder.WriteString(`resource "` + v.Type + `" "` + v.Name + `" {` + "\n")

		for key, value := range v.Attributes {
			builder.WriteString("  " + key + " = " + strconv.Quote(value) + "\n")
		}

		builder.WriteString("}\n\n")
	}

	return os.WriteFile(path, []byte(builder.String()), 0644)
}
