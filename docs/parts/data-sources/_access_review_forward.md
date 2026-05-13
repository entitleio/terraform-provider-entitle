An Entitle Access Review Forward delegates access review responsibilities from one user to another. During periodic access review campaigns, reviewers certify whether users should retain their current access. When a reviewer is unavailable, their review tasks can be forwarded to a designated colleague.

Use this data source to look up an existing access review forward by ID — for example, to inspect its configuration, retrieve the forwarder and target details, or verify a forward is in place for compliance auditing. [Read more about access reviews](https://docs.beyondtrust.com/entitle/docs/access-review).

## Key Concepts

- **Access Review**: A periodic campaign where managers certify that users' access is still appropriate
- **Forwarder**: The reviewer who has delegated their review responsibilities
- **Target**: The user who is handling the forwarded review tasks
- **Active Forward**: A forward persists until the `entitle_access_review_forward` resource is destroyed

## When to Use This Data Source

- Verifying that an access review forward is active (e.g., before a review campaign)
- Retrieving forwarder and target details for compliance documentation
- Auditing which review responsibilities have been delegated and to whom

## Example Usage

### Look Up a Review Forward by ID

```terraform
data "entitle_access_review_forward" "current_delegation" {
  id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

output "review_delegation" {
  value = {
    delegated_from = data.entitle_access_review_forward.current_delegation.forwarder.email
    delegated_to   = data.entitle_access_review_forward.current_delegation.target.email
  }
}
```

### Audit Active Review Forwards

```terraform
data "entitle_access_review_forward" "audit" {
  id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

output "review_delegation_audit" {
  value = {
    id            = data.entitle_access_review_forward.audit.id
    from_user     = data.entitle_access_review_forward.audit.forwarder.email
    to_user       = data.entitle_access_review_forward.audit.target.email
    from_user_id  = data.entitle_access_review_forward.audit.forwarder.id
    to_user_id    = data.entitle_access_review_forward.audit.target.id
  }
}
```

## Query Parameters

### Required

- `id` (String) The unique identifier of the access review forward to retrieve (UUID format).

## Returned Attributes

- `id` (String) The forward's unique identifier (UUID format).
- `forwarder` (Attributes) The reviewer who delegated their responsibilities:
    - `id` (String) The forwarder's unique identifier.
    - `email` (String) The forwarder's email address.
- `target` (Attributes) The user receiving the delegated review tasks:
    - `id` (String) The target's unique identifier.
    - `email` (String) The target's email address.

## Notes

- To create or manage a review forward, use the `entitle_access_review_forward` resource
- Forward IDs can be found in the Entitle UI under **Org Settings** → **Forwards** section
- For access *request* forwards (JIT request task delegation), use `entitle_access_request_forward` instead
