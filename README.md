# terraform-provider-godaddy

[Terraform](https://www.terraform.io/) provider for managing GoDaddy DNS records,
nameservers, and domain registrations.

Built on [`terraform-plugin-framework`](https://github.com/hashicorp/terraform-plugin-framework).
Requires Go 1.26+ and Terraform 1.0+.

## Authentication

Obtain a [GoDaddy API key](https://developer.godaddy.com/keys/) and set it via
the provider attributes or environment variables:

```bash
export GODADDY_API_KEY=...
export GODADDY_API_SECRET=...
```

```terraform
terraform {
  required_providers {
    godaddy = {
      source  = "CruGlobal/godaddy"
      version = "~> 2.0"
    }
  }
}

provider "godaddy" {
  # key and secret may also be supplied via GODADDY_API_KEY / GODADDY_API_SECRET.
}
```

## Resources

| Resource                       | Description                                |
| ------------------------------ | ------------------------------------------ |
| `godaddy_domain_record`        | Manage DNS records on a registered domain. |
| `godaddy_domain_nameservers`   | Manage the nameservers for a domain.       |
| `godaddy_domain_purchase`      | Register and manage a new domain.          |

Per-resource documentation lives under [`docs/resources/`](./docs/resources/)
and is auto-generated from the provider schema via
[`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs). To
regenerate after a schema change, run:

```bash
go generate ./...
```

## Development

```bash
go build ./...
go test ./...
```

If your zone already contains records, make sure your Terraform configuration
covers every existing record — anything not declared will be removed on apply.
The provider also supports `terraform import` for any of its resources, keyed
by the domain name.

## License

Apache 2.0 — see [LICENSE](./LICENSE).
