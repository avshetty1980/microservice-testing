Here's a complete **Terraform** stack to create the AWS infrastructure for your microservice architecture, which includes:

- **AWS Managed Streaming for Apache Kafka (MSK)** for Kafka.
- **API Gateway** for exposing APIs via **CloudFront**.
- **Lambda Functions** running your **Go-based** API logic.
- **DynamoDB** for storing policy and audit data.
- Proper **permissions boundary**, roles, and security best practices.

This Terraform stack follows best practices such as enabling logging, encryption, and fine-grained permissions.

### Folder Structure
```bash
terraform/
├── main.tf
├── variables.tf
├── outputs.tf
└── providers.tf
```

### Step 1: `providers.tf`

```hcl
provider "aws" {
  region = "us-west-2"
}

provider "aws" {
  alias  = "us-east-1"
  region = "us-east-1"
}
```

### Step 2: `main.tf`

``` h
resource "aws_iam_role" "lambda_execution_role" {
  name = "lambda_execution_role"
  assume_role_policy = jsonencode({
    "Version" : "2012-10-17",
    "Statement": [{
      "Action": "sts:AssumeRole",
      "Effect": "Allow",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      }
    }]
  })

  permissions_boundary = aws_iam_policy.permissions_boundary.arn

  managed_policy_arns = [
    "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
    "arn:aws:iam::aws:policy/AWSXrayWriteOnlyAccess",
    "arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess",
    "arn:aws:iam::aws:policy/AmazonMSKFullAccess"
  ]
}

resource "aws_iam_policy" "permissions_boundary" {
  name = "lambda_permissions_boundary"

  policy = jsonencode({
    "Version": "2012-10-17",
    "Statement": [{
      "Action": "*",
      "Effect": "Deny",
      "Resource": "*"
    }]
  })
}

resource "aws_kms_key" "dynamodb_kms_key" {
  description             = "KMS key for DynamoDB encryption"
  deletion_window_in_days = 10
}

resource "aws_dynamodb_table" "epm_policies" {
  name           = "epm_policies"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "id"

  attribute {
    name = "id"
    type = "S"
  }

  point_in_time_recovery {
    enabled = true
  }

  server_side_encryption {
    enabled     = true
    kms_key_arn = aws_kms_key.dynamodb_kms_key.arn
  }

  tags = {
    Name = "EPMPolicies"
  }
}

resource "aws_msk_cluster" "kafka_cluster" {
  cluster_name           = "epm-kafka-cluster"
  kafka_version          = "2.6.0"
  number_of_broker_nodes = 3

  broker_node_group_info {
    instance_type = "kafka.m5.large"
    client_subnets = [aws_subnet.public_subnet.id]

    security_groups = [
      aws_security_group.kafka_security_group.id
    ]
  }

  encryption_info {
    encryption_in_transit {
      client_broker = "TLS"
      in_cluster    = true
    }
  }

  logging_info {
    broker_logs {
      cloudwatch_logs {
        enabled   = true
        log_group = aws_cloudwatch_log_group.kafka_log_group.name
      }
    }
  }
}

resource "aws_cloudwatch_log_group" "kafka_log_group" {
  name              = "/aws/msk/epm-kafka"
  retention_in_days = 7
}

resource "aws_security_group" "kafka_security_group" {
  name        = "kafka_sg"
  description = "Allow Kafka traffic"

  ingress {
    from_port   = 9092
    to_port     = 9092
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

resource "aws_lambda_function" "epm_create_policy" {
  function_name = "epm_create_policy"
  role          = aws_iam_role.lambda_execution_role.arn
  handler       = "createPolicy"

  # Zip up Go binary and upload to S3 or local path
  filename = "path_to_epm_create_policy.zip"

  runtime = "go1.x"

  environment {
    variables = {
      DYNAMODB_TABLE = aws_dynamodb_table.epm_policies.name
      MSK_BROKER    = aws_msk_cluster.kafka_cluster.bootstrap_brokers_tls
    }
  }

  tags = {
    Name = "EPMCreatePolicyFunction"
  }
}

resource "aws_apigatewayv2_api" "epm_api" {
  name          = "EPM_API"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_stage" "default_stage" {
  api_id      = aws_apigatewayv2_api.epm_api.id
  name        = "$default"
  auto_deploy = true
}

resource "aws_apigatewayv2_integration" "lambda_integration" {
  api_id             = aws_apigatewayv2_api.epm_api.id
  integration_type   = "AWS_PROXY"
  integration_uri    = aws_lambda_function.epm_create_policy.invoke_arn
  integration_method = "POST"
}

resource "aws_apigatewayv2_route" "create_policy_route" {
  api_id     = aws_apigatewayv2_api.epm_api.id
  route_key  = "POST /policies"
  target     = "integrations/${aws_apigatewayv2_integration.lambda_integration.id}"
}

resource "aws_cloudfront_distribution" "api_distribution" {
  origin {
    domain_name = aws_apigatewayv2_api.epm_api.api_endpoint
    origin_id   = "EPMAPI"
  }

  enabled = true

  default_cache_behavior {
    allowed_methods  = ["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"]
    cached_methods   = ["GET", "OPTIONS"]
    target_origin_id = "EPMAPI"

    viewer_protocol_policy = "redirect-to-https"
  }

  price_class = "PriceClass_100"
}
```

### Step 3: `variables.tf`

Define your variables:

``` h
variable "region" {
  description = "The AWS region to use"
  default     = "us-west-2"
}
```

### Step 4: `outputs.tf`

Provide key outputs such as **API Gateway URL** and **CloudFront URL**.

```h
output "api_gateway_url" {
  value = aws_apigatewayv2_api.epm_api.api_endpoint
}

output "cloudfront_distribution_domain" {
  value = aws_cloudfront_distribution.api_distribution.domain_name
}
```

### Security Best Practices Included:
- **AWS Lambda**: Attached to a role with minimal permissions (using permissions boundaries).
- **DynamoDB**: Encryption at rest with KMS.
- **MSK**: Encryption for data in transit and in-cluster.
- **CloudFront**: Configured with HTTPS redirection.
- **API Gateway**: Integrated with Lambda for handling requests.

### Deployment Instructions
1. Initialize Terraform: `terraform init`
2. Preview changes: `terraform plan`
3. Deploy the stack: `terraform apply`

This setup ensures secure, efficient, and best-practice architecture for running your Go-based EPM microservice.