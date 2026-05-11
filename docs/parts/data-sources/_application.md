Retrieve a list of all Entitle Applications — the supported application types that can be connected to Entitle as integrations. Applications represent categories of systems (e.g., AWS, GitHub, Slack, PostgreSQL) for which Entitle has built-in connectors and permission management capabilities.

Use this data source to discover available application types, find the correct application `name` value to use when creating an `entitle_integration`, or audit which applications are supported in your Entitle instance.

## Key Concepts

- **Application**: A supported application type with a built-in Entitle connector (e.g., `"aws"`, `"github"`, `"slack"`, `"postgresql"`)
- **Application Name**: The lowercase identifier used in `entitle_integration.application.name` — must match exactly
- **Difference from Integration**: An Application is the *type* of system; an Integration is a specific *instance* of that type (e.g., "GitHub" is the application; "GitHub - My Organization" is the integration)

## When to Use This Data Source

- Discovering available application types before creating an integration
- Finding the exact application `name` string required by `entitle_integration.application.name`
- Listing all supported applications in your Entitle environment
- Validating that a specific application type is available before attempting to create an integration

## Example Usage

### List All Available Applications

```terraform
data "entitle_applications" "all" {}

output "available_applications" {
  value = data.entitle_applications.all.applications[*].name
}
```

### Find the Correct Application Name for an Integration

Use the data source to confirm the exact name before creating an integration:

```terraform
data "entitle_applications" "all" {}

output "application_names" {
  value = data.entitle_applications.all.applications[*].name
}

# Then use the confirmed name in your integration:
resource "entitle_integration" "my_github" {
  name            = "GitHub - My Organization"
  connection_json = jsonencode({
    token        = var.github_token
    organization = var.github_org
  })

  application = {
    name = "github"  # Confirmed from the data source output
  }

  owner    = { id = "7d080bfa-9143-11ee-b9d1-0242ac120001" }
  workflow = { id = "7d080bfa-9143-11ee-b9d1-0242ac120002" }

  allowed_durations       = [3600, 28800]
  allow_creating_accounts = true
}
```

### Output All Supported Application Types

```terraform
data "entitle_applications" "catalog" {}

output "application_catalog" {
  description = "All application types supported by this Entitle instance"
  value       = data.entitle_applications.catalog.applications
}
```

## Query Parameters

This data source has no required or optional parameters — it returns all available applications.

## Returned Attributes

- `applications` (Attributes List) The list of all supported application types:
    - `name` (String) The application's type identifier (lowercase). Use this value in `entitle_integration.application.name`.

## Common Application Names

While the exact list depends on your Entitle configuration, common application names include:

- `"aws"` — Amazon Web Services
- `"github"` — GitHub
- `"slack"` — Slack
- `"okta"` — Okta
- `"google"` — Google Workspace
- `"postgresql"` — PostgreSQL
- `"mongodb"` — MongoDB
- `"pagerduty"` — PagerDuty
- `"opsgenie"` — Opsgenie

Always use the `entitle_applications` data source to confirm available applications in your specific Entitle environment — the list may vary based on your organization's setup.

## Notes

- Application names are case-sensitive and must be lowercase when used in `entitle_integration.application.name`
- This data source returns all applications enabled for your Entitle tenant — contact Entitle support to enable additional integrations
- Refer to the [Entitle integrations documentation](https://docs.beyondtrust.com/entitle/docs/about-entitle-integrations) for the full list of supported applications and their `connection_json` formats
