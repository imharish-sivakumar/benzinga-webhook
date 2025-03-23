variable "region" {
  default = "us-east-1"
}

variable "domain_name" {
  description = "interviewwithhariharan.com"
  default     = "interviewwithhariharan.com"
}

variable "subdomain" {
  description = "The subdomain name, e.g., www."
  default     = "www"
}

variable "vpc_id" {
  default = "vpc-06bc3b438a016658c"
  type    = string
}

variable "subnets" {
  default = ["subnet-079a2adbfaa863a9f", "subnet-086c161a7a0e536e8"]
  type    = list(string)
}