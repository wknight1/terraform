
mock "aws" {
  resource "aws_s3_bucket" {
    defaults = {
      arn = "arn:aws:s3:bucket"
    }

    override {
      addr = aws_s3_bucket.bucket

      values = {
        arn = "arn:aws:s3:provider"
      }
    }
  }
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
