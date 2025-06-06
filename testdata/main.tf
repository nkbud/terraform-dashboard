resource "aws_instance" "web" {
  ami           = "ami-0c55b159cbfafe1d0"
  instance_type = "t2.micro"
  
  tags = {
    Name = "HelloWorld"
    Environment = "dev"
  }
}

resource "aws_security_group" "web_sg" {
  name        = "web_security_group"
  description = "Allow HTTP traffic"
  
  ingress {
    from_port   = 80
    to_port     = 80
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

provider "aws" {
  region = "us-west-2"
}

variable "instance_name" {
  description = "Name of the EC2 instance"
  type        = string
  default     = "HelloWorld"
}

output "instance_ip" {
  description = "Public IP of the instance"
  value       = aws_instance.web.public_ip
}