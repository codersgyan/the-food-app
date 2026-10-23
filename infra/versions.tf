terraform {
  required_version = ">= 1.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 5.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.0"
    }
    http = {
      source  = "hashicorp/http"
      version = "~> 3.0"
    }
  }
}

# Credentials come from the environment: AWS_ACCESS_KEY_ID and
# AWS_SECRET_ACCESS_KEY (or AWS_PROFILE), and CLOUDFLARE_API_TOKEN.
provider "aws" {
  region = var.region

  default_tags {
    tags = { Project = "the-food-app" }
  }
}

provider "cloudflare" {}
