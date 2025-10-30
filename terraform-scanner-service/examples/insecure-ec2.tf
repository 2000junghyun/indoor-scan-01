# Example: Insecure EC2 Instance Configuration

resource "aws_instance" "web" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t2.micro"

  # No encryption for EBS volume - WILL BE FLAGGED
  # Public IP association - WILL BE FLAGGED
  associate_public_ip_address = true

  # Missing monitoring - WILL BE FLAGGED
  monitoring = false

  # Security group with overly permissive rules
  vpc_security_group_ids = [aws_security_group.insecure.id]

  # IMDSv1 enabled - WILL BE FLAGGED
  metadata_options {
    http_tokens = "optional"
  }
}

resource "aws_security_group" "insecure" {
  name        = "insecure-sg"
  description = "Insecure security group"

  # Allow all inbound traffic - WILL BE FLAGGED
  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Allow all outbound traffic
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_ebs_volume" "example" {
  availability_zone = "us-west-2a"
  size              = 40

  # No encryption - WILL BE FLAGGED
  encrypted = false
}
