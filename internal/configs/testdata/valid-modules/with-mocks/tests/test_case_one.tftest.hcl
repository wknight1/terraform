
mock "aws" {
  source = "testing/aws.tfmock.hcl"
}

override {
  addr = aws_s3_bucket.bucket

  values = {
    arn = "arn:aws:s3:file"
  }
}

run "specific" {
  variables {
    bucket_name = "my_test_bucket"
  }

  override {
    addr = aws_s3_bucket.bucket

    values = {
      arn = "arn:aws:s3:run"
    }
  }
}
