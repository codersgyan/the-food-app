output "api_ip" {
  value = aws_instance.api.public_ip
}

output "runner_ip" {
  value = aws_instance.runner.public_ip
}

output "domain" {
  value = var.domain
}

output "ssh_key" {
  value = local_sensitive_file.ssh_key.filename
}
