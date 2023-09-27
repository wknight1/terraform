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
