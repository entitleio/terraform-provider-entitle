An Entitle Access Request Forward delegates access request responsibilities from one user to another. When the original requester is unavailable — for example, due to vacation or leave — their pending access request tasks can be reassigned to a designated colleague.

Use this data source to look up an existing access request forward by ID — for example, to inspect its configuration, retrieve the forwarder and target details, or verify a forward exists as part of a configuration check.

## Key Concepts

- **Forwarder**: The user who has delegated their access request responsibilities
- **Target**: The user who is handling the forwarded tasks
- **Active Forward**: A forward persists until the `entitle_access_request_forward` resource is destroyed

## When to Use This Data Source

- Verifying that a specific access request forward is active
- Retrieving the forwarder and target details for an existing forward
- Auditing current delegation configurations

## Example Usage

### Look Up a Forward by ID

```terraform
data "entitle_access_request_forward" "current_forward" {
  id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

output "forward_details" {
  value = {
    forwarder_email = data.entitle_access_request_forward.current_forward.forwarder.email
    target_email    = data.entitle_access_request_forward.current_forward.target.email
  }
}
```

### Inspect an Existing Forward for Audit

```terraform
data "entitle_access_request_forward" "audit" {
  id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

output "delegation_audit" {
  value = {
    id              = data.entitle_access_request_forward.audit.id
    from_user       = data.entitle_access_request_forward.audit.forwarder.email
    to_user         = data.entitle_access_request_forward.audit.target.email
    from_user_id    = data.entitle_access_request_forward.audit.forwarder.id
    to_user_id      = data.entitle_access_request_forward.audit.target.id
  }
}
```

## Query Parameters

### Required

- `id` (String) The unique identifier of the access request forward to retrieve (UUID format).

## Returned Attributes

- `id` (String) The forward's unique identifier (UUID format).
- `forwarder` (Attributes) The user who delegated their responsibilities:
    - `id` (String) The forwarder's unique identifier.
    - `email` (String) The forwarder's email address.
- `target` (Attributes) The user receiving the delegated tasks:
    - `id` (String) The target's unique identifier.
    - `email` (String) The target's email address.

## Notes

- To create or manage a forward, use the `entitle_access_request_forward` resource
- Forward IDs can be found in the Entitle UI under **Org Settings** → **Forwards** (or Access Request Forwards section)
- For access *review* forwards (periodic review campaigns), use `entitle_access_review_forward` instead
