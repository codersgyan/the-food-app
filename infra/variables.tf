variable "domain" {
  description = "The API's domain. Its DNS zone must be on Cloudflare."
  type        = string
  default     = "food-api.sys0.in"
}

variable "cloudflare_zone" {
  description = "The Cloudflare zone the domain belongs to."
  type        = string
  default     = "sys0.in"
}

variable "region" {
  description = "The AWS region. ap-south-1 is Mumbai."
  type        = string
  default     = "ap-south-1"
}

variable "availability_zone" {
  description = "Both servers go in the same zone, so the network between them is short and steady."
  type        = string
  default     = "ap-south-1a"
}

variable "api_instance_type" {
  description = "The API server: 2 ARM vCPUs and 8 GB of memory."
  type        = string
  default     = "m7g.large"
}

variable "runner_instance_type" {
  description = "The test runner: 4 ARM vCPUs, twice the API server, so the runner is never what slows a test down."
  type        = string
  default     = "c7g.xlarge"
}

variable "disk_gb" {
  description = "Disk size. The first Docker build alone takes about 3 GB."
  type        = number
  default     = 20
}

variable "lifetime_hours" {
  description = "Each server deletes itself after this many hours, in case you forget make cloud-down."
  type        = number
  default     = 4
}
