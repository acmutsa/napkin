package compiler

import (
	"os"
	"strings"
	"testing"
)

func TestWriteTerraformFile(t *testing.T) {
	tf := &TFFile{
		Resources: []TFResource{
			{
				Type: "aws_instance",
				Name: "web",
				Attributes: map[string]string{
					"ami": "ami-123",
				},
			},
		},
	}

	tmp, err := os.CreateTemp("", "test-*.tf")
	if err != nil {
		t.Fatal(err)
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	err = WriteTerraformFile(tf, tmp.Name())
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}

	output := string(data)

	if !strings.Contains(output, `resource "aws_instance" "web"`) {
		t.Errorf("missing resource declaration")
	}

	if !strings.Contains(output, `ami = "ami-123"`) {
		t.Errorf("missing attribute")
	}
}
