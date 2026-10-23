# Two servers in one AWS zone: the API server, and a test runner that sends
# the load. See infra/README.md.

# A new SSH key for these servers. The private half is saved in
# infra/secrets/, which git ignores.
resource "tls_private_key" "ssh" {
  algorithm = "ED25519"
}

resource "local_sensitive_file" "ssh_key" {
  content         = tls_private_key.ssh.private_key_openssh
  filename        = "${path.module}/secrets/ssh-key.pem"
  file_permission = "0600"
}

resource "aws_key_pair" "this" {
  key_name_prefix = "the-food-app-"
  public_key      = tls_private_key.ssh.public_key_openssh
}

# Only your current IP address can log in with SSH.
data "http" "my_ip" {
  url = "https://checkip.amazonaws.com"
}

locals {
  my_cidr = "${chomp(data.http.my_ip.response_body)}/32"
}

data "aws_vpc" "default" {
  default = true
}

data "aws_subnet" "default" {
  vpc_id            = data.aws_vpc.default.id
  availability_zone = var.availability_zone
  default_for_az    = true
}

# The latest Ubuntu 26.04 for ARM, as Canonical publishes it.
data "aws_ssm_parameter" "ubuntu" {
  name = "/aws/service/canonical/ubuntu/server/26.04/stable/current/arm64/hvm/ebs-gp3/ami-id"
}

resource "aws_security_group" "api" {
  name_prefix = "the-food-app-api-"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "SSH, from your IP only"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [local.my_cidr]
  }
  ingress {
    description = "HTTP: Let's Encrypt's check, and the redirect to HTTPS"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    description = "HTTPS: the app"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "runner" {
  name_prefix = "the-food-app-runner-"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "SSH, from your IP only"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [local.my_cidr]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

locals {
  # The safety net: the server shuts itself down after lifetime_hours, and
  # because of instance_initiated_shutdown_behavior, AWS then deletes it.
  self_destruct = <<-EOT
    #!/bin/bash
    shutdown -h +${var.lifetime_hours * 60}
  EOT
}

resource "aws_instance" "api" {
  ami                                  = data.aws_ssm_parameter.ubuntu.insecure_value
  instance_type                        = var.api_instance_type
  subnet_id                            = data.aws_subnet.default.id
  vpc_security_group_ids               = [aws_security_group.api.id]
  key_name                             = aws_key_pair.this.key_name
  user_data                            = local.self_destruct
  instance_initiated_shutdown_behavior = "terminate"

  root_block_device {
    volume_size = var.disk_gb
    volume_type = "gp3"
  }

  tags = { Name = "the-food-app-api" }
}

resource "aws_instance" "runner" {
  ami                                  = data.aws_ssm_parameter.ubuntu.insecure_value
  instance_type                        = var.runner_instance_type
  subnet_id                            = data.aws_subnet.default.id
  vpc_security_group_ids               = [aws_security_group.runner.id]
  key_name                             = aws_key_pair.this.key_name
  user_data                            = local.self_destruct
  instance_initiated_shutdown_behavior = "terminate"

  root_block_device {
    volume_size = var.disk_gb
    volume_type = "gp3"
  }

  tags = { Name = "the-food-app-runner" }
}

# Point the domain at the API server. It's DNS only (not proxied by
# Cloudflare), so nginx on our server is what answers.
data "cloudflare_zone" "this" {
  filter = { name = var.cloudflare_zone }
}

resource "cloudflare_dns_record" "api" {
  zone_id = data.cloudflare_zone.this.zone_id
  name    = var.domain
  type    = "A"
  content = aws_instance.api.public_ip
  ttl     = 60
  proxied = false
}
