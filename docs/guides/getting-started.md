---
page_title: "Getting Started - TrueNAS Provider"
subcategory: "Guides"
description: |-
  A step-by-step guide to getting started with the TrueNAS Terraform provider.
---

# Getting Started

This guide walks you through setting up the TrueNAS Terraform provider from scratch and creating your first resources.

## Prerequisites

- TrueNAS SCALE 24.04 or later
- Terraform 1.5 or later installed locally
- Network access to your TrueNAS instance

## Step 1: Create a TrueNAS API Key

1. Log in to the TrueNAS web interface.
2. Navigate to **Credentials → API Keys**.
3. Click **Add** and give your key a descriptive name (e.g., `terraform`).
4. Copy the generated API key — it is only shown once.

~> **Security Note:** Store this API key in a secrets manager or Terraform variable. Never commit it to version control.

## Step 2: Configure the Provider

Create a new directory for your Terraform configuration:

```bash
mkdir truenas-infra && cd truenas-infra
```

Create `versions.tf`:

```terraform
terraform {
  required_version = ">= 1.5"
  required_providers {
    truenas = {
      source  = "registry.terraform.io/salty/truenas"
      version = "~> 0.1"
    }
  }
}
```

Create `provider.tf`:

```terraform
provider "truenas" {
  url     = var.truenas_url
  api_key = var.truenas_api_key
}
```

Create `variables.tf`:

```terraform
variable "truenas_url" {
  description = "The URL of the TrueNAS SCALE instance"
  type        = string
}

variable "truenas_api_key" {
  description = "TrueNAS API key"
  type        = string
  sensitive   = true
}
```

Create `terraform.tfvars` (keep this out of version control):

```hcl
truenas_url     = "https://truenas.example.com"
truenas_api_key = "your-api-key-here"
```

Add `terraform.tfvars` to your `.gitignore`:

```
terraform.tfvars
*.tfvars
.terraform/
.terraform.lock.hcl
```

Alternatively, use environment variables:

```bash
export TRUENAS_URL="https://truenas.example.com"
export TRUENAS_API_KEY="your-api-key-here"
```

## Step 3: Create Your First Dataset

Create `main.tf`:

```terraform
resource "truenas_dataset" "first" {
  pool        = "tank"
  name        = "terraform-managed"
  compression = "LZ4"
  atime       = "OFF"
  comments    = "Created and managed by Terraform"
}

output "dataset_path" {
  value = truenas_dataset.first.id
}

output "mount_point" {
  value = truenas_dataset.first.mount_point
}
```

## Step 4: Initialize and Apply

Initialize Terraform and download the provider:

```bash
terraform init
```

Review what will be created:

```bash
terraform plan
```

Apply the configuration:

```bash
terraform apply
```

Type `yes` when prompted. Terraform will create the dataset and display the outputs:

```
Outputs:

dataset_path = "tank/terraform-managed"
mount_point  = "/mnt/tank/terraform-managed"
```

## Step 5: Verify in TrueNAS

Log in to the TrueNAS UI and navigate to **Storage → Datasets** to confirm `tank/terraform-managed` was created with the correct properties.

## Step 6: Add an NFS Share

Add the following to `main.tf`:

```terraform
resource "truenas_share_nfs" "first" {
  path    = truenas_dataset.first.mount_point
  comment = "NFS share for terraform-managed dataset"
  enabled = true
}
```

Run `terraform apply` again to create the share.

## Cleaning Up

To remove all resources created by this guide:

```bash
terraform destroy
```

## Next Steps

- See the [Kubernetes Storage Guide](kubernetes-storage.md) for using TrueNAS as persistent storage for Kubernetes
- See the [Backup Strategy Guide](backup-strategy.md) for automating snapshots and replication
- See the [Importing Existing Resources Guide](importing-existing.md) to bring existing TrueNAS configuration under Terraform management
