# 1Password Secret Terraform Provider

This repository contains the code of the terraform provider, allowing to resolve [1Password secret references](https://developer.1password.com/docs/cli/secret-reference-syntax/)
into their respective secret values, which then can be used in other terraform resources.

This approach both leverages efficiency, as it allows secrets to be managed in one single place and reduces risk of 
error-prone copy pasting of secret values back and forth.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up-to-date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

To use this provider add the following snippets to your `provider.tf` file:
```terraform
terraform {
  required_providers {
    opsecret = {
      source = "registry.terraform.io/qaware-internal/onepassword-secret"
    }
    ...
  }
}

provider "opsecret" {
  # provide a service account token directly
  # if omitted, the OP_SERVICE_ACCOUNT_TOKEN environment variable will be used instead.
  service_account_token = "op_s3cr3t"
}

```

To resolve and use a secret value stored in 1Password use the following snippet:
```terraform
data "opsecret_secret_reference" "secret_reference" {
  id = "op://vault-name/item-name/section-name/field-name"
}

resource "whatever" "some_resource" {
  attribute = data.opsecret_secret_reference.secret_reference.value
}
```

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To use the compiled provider in a local repository, add a `dev_overrides` directive in your terraform / opentofu configuration file (see [official Documentation](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers) for details). 

To generate or update documentation, run `make generate`.