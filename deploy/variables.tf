variable "aws_region" {
  description = "AWS region to deploy into"
  type        = string
  default     = "eu-west-2"
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.micro"
}

variable "project_name" {
  description = "Project name used for resource naming and tagging"
  type        = string
  default     = "invoice-bot"
}

variable "github_repo" {
  description = "GitHub repository in the format owner/repo (used for OIDC trust policy)"
  type        = string
  default     = "sheriff4000/invoice-bot"
}
