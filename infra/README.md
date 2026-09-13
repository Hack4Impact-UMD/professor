# Professor Infrastructure

Terraform configuration for the Google Cloud resources used by Professor.

The Terraform code manages the following: 
- Cloud Run service
- Cloud Tasks queue
- Artifact Registry repository
- Service accounts and IAM bindings
- Secret Manager secret metadata
- Monitoring alert policies

It does not manage: 
- Resources shared with the App Portal, which are managed by Firebase
- Container image revisions and builds, which are managed by the GitHub Actions deployment workflow
- Secret Manager secret values, which are managed manually

## Local set-up

Install:

- Terraform
- Google Cloud CLI

Authenticate with Google Cloud:

```bash
gcloud auth login
gcloud auth application-default login
```

Set the active project:

```bash
gcloud config set project h4i-applications
```

Initialize Terraform state from this directory (`infra/`) with:

```bash
terraform init
```

## Planning and applying changes

Once you have changes, diff against current state with:

```bash
terraform plan
```

The GitHub Actions workflow applies merged changes. Avoid applying changes manually. 
