Retrieve every Entitle Integration matching a filter. Use this data source to discover the integrations in your organization, look them up by ID, name, owner, maintainer, workflow or application — for example to feed their IDs into `entitle_resources` or `entitle_roles`.

An integration is a configured connection to a specific instance of an application (e.g. a particular AWS account, GitHub organization or Okta tenant). Use `entitle_integration` (singular) to read the full configuration of one integration. [Read more about integrations](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).

## Key Concepts

- **All results**: The data source always returns every matching integration. The provider pages through the API with a cursor, so no integration is skipped or repeated while the list changes. Without a filter, every integration is returned
- **ids**: `filter.ids` returns specific integrations by ID, up to 100; with `ids_operator = "isNot"` it returns every integration except them
- **names**: Matches integration names case-insensitively by substring; `names_operator` and `names_relation` control how several values are matched. Values cannot contain a comma
- **Filters**: `requestable`, `workflow_ids`, `owner_ids`, `maintainer_ids` and `application_ids` narrow the results; all filters are combined with AND
- **Difference from entitle_integration**: `entitle_integrations` (plural) returns only `id`, `name` and `application` per integration; use `entitle_integration` (singular) for full details

## Example Usage

### List Integrations

```terraform
data "entitle_integrations" "all" {}

output "integration_names" {
  value = data.entitle_integrations.all.integrations[*].name
}
```

### Find Integrations by Name

```terraform
data "entitle_integrations" "aws_prod" {
  filter {
    names          = ["aws", "prod"]
    names_relation = "and" # names containing both
  }
}
```

### Integrations of an Application Owned by a User

```terraform
data "entitle_integrations" "owned_okta" {
  filter {
    application_ids = ["7d080bfa-9143-11ee-b9d1-0242ac120010"]
    owner_ids       = ["7d080bfa-9143-11ee-b9d1-0242ac120011"]
    requestable     = true
  }
}
```

### Look Up Specific Integrations by ID

```terraform
data "entitle_integrations" "selected" {
  filter {
    ids = [
      "7d080bfa-9143-11ee-b9d1-0242ac120001",
      "7d080bfa-9143-11ee-b9d1-0242ac120002",
    ]
  }
}
```

### Every Integration Except Some

```terraform
data "entitle_integrations" "others" {
  filter {
    ids          = ["7d080bfa-9143-11ee-b9d1-0242ac120001"]
    ids_operator = "isNot"
  }
}
```

## Notes

- Results are sorted by ID
