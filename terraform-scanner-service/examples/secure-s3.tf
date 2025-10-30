# Example: Secure S3 Bucket Configuration

resource "aws_s3_bucket" "secure_example" {
  bucket = "my-secure-bucket"
}

resource "aws_s3_bucket_encryption" "secure_example" {
  bucket = aws_s3_bucket.secure_example.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.example.arn
    }
  }
}

resource "aws_s3_bucket_versioning" "secure_example" {
  bucket = aws_s3_bucket.secure_example.id
  
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_public_access_block" "secure_example" {
  bucket = aws_s3_bucket.secure_example.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_logging" "secure_example" {
  bucket = aws_s3_bucket.secure_example.id

  target_bucket = aws_s3_bucket.log_bucket.id
  target_prefix = "log/"
}

resource "aws_s3_bucket" "log_bucket" {
  bucket = "my-log-bucket"
}

resource "aws_kms_key" "example" {
  description             = "KMS key for S3 encryption"
  deletion_window_in_days = 10
}
