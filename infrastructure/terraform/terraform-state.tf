# DynamoDB table for Terraform state locking
# This should be created before running main terraform configuration

resource "aws_dynamodb_table" "terraform_state_lock" {
  name           = "terraform-state-lock"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "LockID"

  attribute {
    name = "LockID"
    type = "S"
  }

  server_side_encryption {
    enabled     = true
    kms_key_arn = aws_kms_key.dynamodb.arn
  }

  point_in_time_recovery {
    enabled = true
  }

  tags = merge(local.common_tags, {
    Name = "terraform-state-lock"
  })
}

# KMS key for DynamoDB encryption
resource "aws_kms_key" "dynamodb" {
  description                        = "KMS key for DynamoDB table encryption"
  deletion_window_in_days           = 7
  key_usage                         = "ENCRYPT_DECRYPT"
  customer_master_key_spec          = "SYMMETRIC_DEFAULT"
  enable_key_rotation               = true
  multi_region                      = false

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action   = "kms:*"
        Resource = "*"
      },
      {
        Sid    = "Allow DynamoDB Service"
        Effect = "Allow"
        Principal = {
          Service = "dynamodb.amazonaws.com"
        }
        Action = [
          "kms:Decrypt",
          "kms:DescribeKey",
          "kms:Encrypt",
          "kms:GenerateDataKey*",
          "kms:ReEncrypt*"
        ]
        Resource = "*"
      }
    ]
  })

  tags = merge(local.common_tags, {
    Name = "${local.cluster_name}-dynamodb-kms-key"
  })
}

resource "aws_kms_alias" "dynamodb" {
  name          = "alias/${local.cluster_name}-dynamodb"
  target_key_id = aws_kms_key.dynamodb.key_id
}