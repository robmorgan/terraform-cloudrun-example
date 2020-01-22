# Terraform Serverless CI/CD Pipeline Example

This repo contains an example of deploying a serverless CI/CD pipeline on GCP using Terraform. You can read
more about it on the associated blog post - "[Deploy a Serverless CI/CD Pipeline on GCP using Cloud Run, Cloud Build & Terraform](https://robmorgan.id.au/posts/deploy-a-serverless-cicd-pipeline-on-gcp-using-cloud-run-and-terraform/)".

![GCP Serverless CI/CD Pipeline Architecture](https://github.com/robmorgan/terraform-cloudrun-example/blob/master/_docs/gcp-serverless-cicd-pipeline.png)

## Features

- Enable and configure the relevant APIs and IAM permissions
- Deploy a Git repo using Cloud Source Repositories
- Deploy a Cloud Build Trigger
- Deploy a Cloud Run service

## Contributions

Contributions to this repo are very welcome and appreciated! If you find a bug or want to add a new feature or
even contribute an entirely new module, I am very happy to accept pull requests, provide feedback, and run your
changes through the automated test suite.

## License

Please see [LICENSE](https://github.com/robmorgan/terraform-cloudrun-example/blob/master/LICENSE) for details on how the code in this repo is licensed.

Copyright &copy; 2020 Rob Morgan.
