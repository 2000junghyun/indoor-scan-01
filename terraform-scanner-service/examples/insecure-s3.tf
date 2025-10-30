# Example: Insecure S3 Bucket Configuration

resource "aws_s3_bucket" "example" {
  bucket = "my-insecure-bucket"
  
  # Missing encryption configuration - WILL BE FLAGGED
  # Missing versioning - WILL BE FLAGGED
  # Missing access logging - WILL BE FLAGGED
}

resource "aws_s3_bucket_public_access_block" "example" {
  bucket = aws_s3_bucket.example.id

  # Public access not blocked - WILL BE FLAGGED
  block_public_acls       = false
  block_public_policy     = false
  ignore_public_acls      = false
  restrict_public_buckets = false
}

resource "aws_s3_bucket_acl" "example" {
  bucket = aws_s3_bucket.example.id
  acl    = "public-read"  # Public ACL - WILL BE FLAGGED
}
