terraform {
  required_version = ">= 1.0.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  backend "s3" {
    bucket         = "my-terraform-state-bucket"
    key            = "nanovms/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "my-terraform-lock-table"
    encrypt        = true
  }
}

provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"
}

provider "aws" {
  alias  = "eu_west_1"
  region = "eu-west-1"
}

provider "aws" {
  alias  = "ap_southeast_1"
  region = "ap-southeast-1"
}

module "us_east_1_cluster" {
  source = "./modules/sandbox-cluster"
  providers = {
    aws = aws.us_east_1
  }
  environment        = var.environment
  region             = "us-east-1"
  vpc_cidr           = "10.0.0.0/16"
  ecr_repository_url = aws_ecr_repository.us_east_1.repository_url
}

module "eu_west_1_cluster" {
  source = "./modules/sandbox-cluster"
  providers = {
    aws = aws.eu_west_1
  }
  environment        = var.environment
  region             = "eu-west-1"
  vpc_cidr           = "10.1.0.0/16"
  ecr_repository_url = aws_ecr_repository.eu_west_1.repository_url
}

module "ap_southeast_1_cluster" {
  source = "./modules/sandbox-cluster"
  providers = {
    aws = aws.ap_southeast_1
  }
  environment        = var.environment
  region             = "ap-southeast-1"
  vpc_cidr           = "10.2.0.0/16"
  ecr_repository_url = aws_ecr_repository.ap_southeast_1.repository_url
}

# ECR Repositories
resource "aws_ecr_repository" "us_east_1" {
  name                 = "nanovms-sandbox"
  image_tag_mutability = "MUTABLE"
  provider             = aws.us_east_1
}

resource "aws_ecr_repository" "eu_west_1" {
  name                 = "nanovms-sandbox"
  image_tag_mutability = "MUTABLE"
  provider             = aws.eu_west_1
}

resource "aws_ecr_repository" "ap_southeast_1" {
  name                 = "nanovms-sandbox"
  image_tag_mutability = "MUTABLE"
  provider             = aws.ap_southeast_1
}

# Route53
resource "aws_route53_zone" "main" {
  name    = var.domain_name
  provider = aws.us_east_1
}

resource "aws_route53_record" "us_east_1" {
  zone_id = aws_route53_zone.main.zone_id
  name    = "us.${var.domain_name}"
  type    = "A"
  ttl     = 300
  records = [module.us_east_1_cluster.nlb_dns_name]
  provider = aws.us_east_1
}

resource "aws_route53_record" "eu_west_1" {
  zone_id = aws_route53_zone.main.zone_id
  name    = "eu.${var.domain_name}"
  type    = "A"
  ttl     = 300
  records = [module.eu_west_1_cluster.nlb_dns_name]
  provider = aws.eu_west_1
}

resource "aws_route53_record" "ap_southeast_1" {
  zone_id = aws_route53_zone.main.zone_id
  name    = "ap.${var.domain_name}"
  type    = "A"
  ttl     = 300
  records = [module.ap_southeast_1_cluster.nlb_dns_name]
  provider = aws.ap_southeast_1
}
