output "ec2_public_ip" {
  description = "Public IP address of the EC2 instance"
  value       = aws_instance.bot.public_ip
}

output "ssh_command" {
  description = "SSH command to connect to the EC2 instance"
  value       = "ssh -i ${local_file.ssh_private_key.filename} ec2-user@${aws_instance.bot.public_ip}"
}

output "ecr_repository_url" {
  description = "Full ECR repository URL for the invoice-bot image"
  value       = aws_ecr_repository.bot.repository_url
}

output "github_actions_role_arn" {
  description = "IAM role ARN for GitHub Actions to assume via OIDC"
  value       = aws_iam_role.github_actions.arn
}

output "aws_account_id" {
  description = "AWS account ID (needed for GitHub secrets)"
  value       = local.account_id
}

output "aws_region" {
  description = "AWS region (needed for GitHub secrets)"
  value       = var.aws_region
}
