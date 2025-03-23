# Specify the provider
provider "aws" {
  region = "us-east-1" # Replace with your preferred AWS region
}

#S3 Bucket to store the file in persistant manner

terraform {
  backend "s3" {
    bucket  = "generic-infra-bucket"     # Replace with your S3 bucket name
    key     = "terraform/state/filepath" # Replace with the desired state file path
    region  = "us-east-1"                # Replace with your bucket region
    encrypt = true                       # Encrypt state file
  }
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.84.0" # Specify your version constraint here
    }
  }
}

# Create a key pair (replace with your key pair details if needed)
resource "aws_key_pair" "my_key" {
  key_name   = "my-key"                  # Replace with your desired key name
  public_key = file("~/.ssh/id_rsa.pub") # Path to your public key
}

# Define a security group
resource "aws_security_group" "allow_ssh_http" {
  name        = "allow_ssh_http"
  description = "Allow SSH and HTTP traffic"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] # Allow SSH from anywhere
  }

  ingress {
    from_port       = 80
    to_port         = 80
    protocol        = "tcp"
    security_groups = [aws_security_group.elb_sg.id] # Reference the ELB SG
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"] # Allow all outbound traffic
  }
}

# Launch an EC2 instance
resource "aws_instance" "dev_trial_instance" {
  ami               = "ami-0885b1f6bd170450c" # Ubuntu 20.04 LTS AMI (Free Tier eligible)
  instance_type     = "t2.micro"              # Free tier instance type
  availability_zone = "us-east-1a"
  key_name          = aws_key_pair.my_key.key_name
  security_groups   = [aws_security_group.allow_ssh_http.name]

  tags = {
    Name = "dev_trial_instance"
  }
}


# Route 53 Hosted Zone
data "aws_route53_zone" "primary_zone" {
  name = var.domain_name
}

# ACM Certificate
resource "aws_acm_certificate" "ssl_cert" {
  domain_name       = var.domain_name
  validation_method = "DNS"

  subject_alternative_names = [
    "${var.subdomain}.${var.domain_name}",
    var.domain_name
  ]

  lifecycle {
    create_before_destroy = true
  }
}


# DNS Validation Record
resource "aws_route53_record" "validation_record" {
  for_each = {
    for dvo in aws_acm_certificate.ssl_cert.domain_validation_options : dvo.domain_name => {
      name  = dvo.resource_record_name
      type  = dvo.resource_record_type
      value = dvo.resource_record_value
    }
  }

  zone_id = data.aws_route53_zone.primary_zone.zone_id

  name    = each.value.name
  type    = each.value.type
  records = [each.value.value]
  ttl     = 60
}

# Wait for ACM Validation
resource "aws_acm_certificate_validation" "ssl_validation" {
  certificate_arn         = aws_acm_certificate.ssl_cert.arn
  validation_record_fqdns = [for record in aws_route53_record.validation_record : record.fqdn]
}

# Security Group for Load Balancer
resource "aws_security_group" "elb_sg" {
  name        = "elb-sg"
  description = "Security group for ELB"
  vpc_id      = var.vpc_id # Replace with your VPC ID

  ingress {
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

# Application Load Balancer (ALB)
resource "aws_lb" "app_lb" {
  name               = "portfolio-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.elb_sg.id]
  subnets            = var.subnets # Replace with public subnets

  enable_deletion_protection = false

  access_logs {
    bucket  = "generic-infra-bucket"
    enabled = true
    prefix  = "alb-logs"
  }
}

# Target Group
resource "aws_lb_target_group" "app_tg" {
  name     = "portfolio-target-group"
  port     = 80
  protocol = "HTTP"
  vpc_id   = var.vpc_id # Replace with your VPC ID

  health_check {
    path                = "/health"
    interval            = 30
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 2
  }
}

# Listener for HTTPS with ACM Certificate
resource "aws_lb_listener" "https_listener" {
  load_balancer_arn = aws_lb.app_lb.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = aws_acm_certificate.ssl_cert.arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.app_tg.arn
  }
}

# Attach EC2 Instance to Target Group
resource "aws_lb_target_group_attachment" "tg_attachment" {
  target_group_arn = aws_lb_target_group.app_tg.arn
  target_id        = aws_instance.dev_trial_instance.id
  port             = 80
}

# Route 53 Records for the Domain
resource "aws_route53_record" "root_domain" {
  zone_id = data.aws_route53_zone.primary_zone.zone_id
  name    = var.domain_name
  type    = "A"

  alias {
    name                   = aws_lb.app_lb.dns_name
    zone_id                = aws_lb.app_lb.zone_id
    evaluate_target_health = true
  }
}

resource "aws_route53_record" "www_subdomain" {
  zone_id = data.aws_route53_zone.primary_zone.zone_id
  name    = "${var.subdomain}.${var.domain_name}"
  type    = "A"

  alias {
    name                   = aws_lb.app_lb.dns_name
    zone_id                = aws_lb.app_lb.zone_id
    evaluate_target_health = true
  }
}

resource "aws_s3_bucket_policy" "alb_log_policy" {
  bucket = "generic-infra-bucket"

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Sid    = "AWSLogDeliveryWrite",
        Effect = "Allow",
        Principal = {
          Service = "elasticloadbalancing.amazonaws.com"
        },
        Action   = "s3:PutObject",
        Resource = "arn:aws:s3:::generic-infra-bucket/alb-logs/AWSLogs/*",
        Condition = {
          StringEquals = {
            "s3:x-amz-acl" = "bucket-owner-full-control"
          }
        }
      },
      {
        Sid    = "AWSLogDeliveryAclCheck",
        Effect = "Allow",
        Principal = {
          Service = "elasticloadbalancing.amazonaws.com"
        },
        Action   = "s3:GetBucketAcl",
        Resource = "arn:aws:s3:::generic-infra-bucket"
      }
    ]
  })
}


output "aws_ec2_instance_public_dns" {
  value = aws_instance.dev_trial_instance.public_dns
}