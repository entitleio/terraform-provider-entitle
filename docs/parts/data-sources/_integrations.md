Retrieve a list of Entitle Integrations. Use this data source to discover the integrations in your organization, look up specific integrations by ID, or page through all of them — for example to feed their IDs into `entitle_resources` or `entitle_roles`.

An integration is a configured connection to a specific instance of an application (e.g. a particular AWS account, GitHub organization or Okta tenant). Use `entitle_integration` (singular) to read the full configuration of one integration. [Read more about integrations](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).

## Key Concepts

- **No filter**: Without a filter, the first page of integrations is returned
- **ids**: `filter.ids` returns specific integrations by ID, up to 100; with `ids_operator = "isNot"` it returns every integration except them
- **after_id**: A cursor for reading the whole list page by page without skipping or repeating items while it changes. Cannot be combined with a `page` other than 1
- **Pagination**: Use `page` and `per_page` (at most 100 results per page)
- **Difference from entitle_integration**: `entitle_integrations` (plural) returns only `id`, `name` and `application` per integration; use `entitle_integration` (singular) for full details

## Example Usage

### List Integrations

```terraform
data "entitle_integrations" "all" {}

output "integration_names" {
  value = data.entitle_integrations.all.integrations[*].name
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

### Paginate with a Cursor

Pass the ID of the last integration of the previous page as `after_id`:

```terraform
data "entitle_integrations" "next_page" {
  filter {
    after_id = "7d080bfa-9143-11ee-b9d1-0242ac120099"
    per_page = 100
  }
}
```

## Notes

- `integrations` contains only the requested page (at most 100 results)
- With `after_id`, results are sorted by ID; stop when a page returns fewer than `per_page` integrations
