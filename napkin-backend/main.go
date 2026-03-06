package main

import (
	"fmt"
	"log"
	"napkin-backend/compiler"
	"net/http"

	"napkin-backend/lib/handlers"
	"napkin-backend/lib/middleware"
	"napkin-backend/lib/static"
)

func main() {
	ir := compiler.IR{
		Nodes: []compiler.GraphNode{
			{
				ID:    "aws",
				Class: compiler.ClassProvider,
				Type:  "aws",
				Attributes: map[string]string{
					"region": "us-east-1",
				},
			},
			{
				ID:    "web",
				Class: compiler.ClassResource,
				Type:  "aws_instance",
				Attributes: map[string]string{
					"ami":           "ami-123",
					"instance_type": "t2.micro",
				},
			},
		},
	}

	target := &compiler.TerraformTarget{}

	tfFile, err := target.Compile(ir)
	if err != nil {
		panic(err)
	}

	err = compiler.WriteTerraformFile(tfFile, "output.tf")
	if err != nil {
		panic(err)
	}

	fmt.Println("Terraform file written to output.tf")

	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("GET /health", handlers.HealthHandler)
	mux.HandleFunc("GET /api/hello", handlers.HelloHandler)

	// Serve static files in production
	static.SetupStaticHandler(mux)

	// Enable CORS middleware (only needed in development when not using proxy)
	handler := middleware.CORS(mux)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
