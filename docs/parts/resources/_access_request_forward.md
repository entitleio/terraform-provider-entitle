An Entitle Access Request Forward delegates access request responsibilities from one user to another. When the original requester is unavailable — for example, due to vacation, leave, or a role change — their pending access request tasks can be reassigned to a designated colleague.

This is useful in scenarios where users have pending approval actions or request tasks that must be handled in their absence, ensuring that access workflows are not blocked by individual unavailability.

## Key Concepts

- **Forwarder**: The user who is delegating their access request responsibilities (the one who is away or unavailable)
- **Target**: The user who will receive and handle the forwarded access request tasks
- **Scope**: All active access request responsibilities of the forwarder are redirected to the target for the duration of the forward

## When to Use Access Request Forwards

- An employee going on vacation who has pending access requests awaiting their action
- A team lead being replaced by a colleague while on leave
- An organizational restructuring where request responsibilities need to be reassigned
- Automating delegation rules for known planned absences (e.g., pre-configuring forwards before a holiday period)

## Example Usage

### Basic Access Request Forward

Delegate all of Alice's access request tasks to Bob:

```terraform
resource "entitle_access_request_forward" "alice_to_bob" {
  forwarder = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"  # Alice's user ID
  }
  target = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"  # Bob's user ID
  }
}
```

### Using Email Instead of ID

Reference users by email address rather than UUID:

```terraform
resource "entitle_access_request_forward" "vacation_forward" {
  forwarder = {
    email = "alice@example.com"
  }
  target = {
    email = "bob@example.com"
  }
}
```

### Data-Driven Forward Using Data Sources

Look up users dynamically and create a forward:

```terraform
data "entitle_user" "forwarder" {
  email = "alice@example.com"
}

data "entitle_user" "delegate" {
  email = "bob@example.com"
}

resource "entitle_access_request_forward" "planned_absence" {
  forwarder = {
    id = data.entitle_user.forwarder.id
  }
  target = {
    id = data.entitle_user.delegate.id
  }
}
```

### Multiple Forwards for a Team

Set up forwards for multiple team members during a planned event:

```terraform
locals {
  vacation_delegates = {
    "alice@example.com" = "charlie@example.com"
    "bob@example.com"   = "diana@example.com"
  }
}

data "entitle_users" "all_users" {}

resource "entitle_access_request_forward" "team_vacation" {
  for_each = local.vacation_delegates

  forwarder = {
    email = each.key
  }
  target = {
    email = each.value
  }
}
```

## Attributes Reference

### Required

- `forwarder` (Attributes) The user who is delegating their access request responsibilities. See [forwarder / target](#forwarder--target) below.
- `target` (Attributes) The user who will receive and handle the forwarded tasks. See [forwarder / target](#forwarder--target) below.

### Read-Only

- `id` (String) The unique identifier of the access request forward (UUID format).

### forwarder / target

Both `forwarder` and `target` accept the same attributes:

- `id` (Optional, String) The user's unique identifier (UUID format). Use this for performance-critical or programmatic configurations. Obtain from the `entitle_user` data source.
- `email` (Optional, String) The user's email address. Can be used as an alternative to `id` — useful when UUIDs are not easily available.

**Note:** At least one of `id` or `email` should be provided. If both are provided, `id` takes precedence.

## Import

Existing access request forwards can be imported using their UUID:

```shell
terraform import entitle_access_request_forward.example a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

### Finding the Forward ID

Access Request Forward IDs can be retrieved via the `entitle_access_request_forward` data source:

```terraform
data "entitle_access_request_forward" "existing" {
  id = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
}
```

## Notes and Best Practices

### Lifecycle Management

- Access request forwards created via Terraform persist until explicitly destroyed with `terraform destroy` or the resource is removed from configuration
- Consider using Terraform lifecycle rules or time-limited automation to remove forwards after a planned absence ends

### Forward Behavior

- All active access request responsibilities of the forwarder are redirected to the target
- The forward is not time-bounded by default — create and destroy it as needed to align with the absence period
- Multiple forwards can exist simultaneously (a user can delegate to multiple targets, or multiple users can delegate to the same target)

### Difference from Access Review Forward

- **Access Request Forward** (`entitle_access_request_forward`): Delegates responsibility for *access requests* — i.e., approving or managing incoming JIT access requests
- **Access Review Forward** (`entitle_access_review_forward`): Delegates responsibility for *access reviews* — i.e., reviewing and certifying existing access during periodic access review campaigns

Use the correct resource type based on what kind of tasks need to be delegated.
