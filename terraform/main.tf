terraform {
  required_version = ">= 1.5"
}

provider "aws" {
  region = var.aws_region
}

variable "aws_region" {
  default = "us-west-2"
}

variable "environment" {
  type = string
}

module "cluster" {
  source      = "./modules/cluster"
  environment = var.environment
  region      = var.aws_region
}
