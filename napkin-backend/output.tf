terraform {
  required_providers {
    aws = {
      version = "~> 5.0"
      source = "hashicorp/aws"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

resource "aws_instance" "web" {
  ami = "ami-123"
  instance_type = "t2.micro"
}

