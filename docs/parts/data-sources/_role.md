An Entitle Role defines a specific permission level within a resource — it represents the most granular unit of access in Entitle, mapping to a real access level inside an integrated application (e.g., "read-only", "contributor", or "admin" within a specific AWS account, GitHub repository, or database).

Use this data source to look up an existing role by ID. This is useful for referencing a specific role when building policies, bundles, or prerequisite permission configurations without needing to manage the role itself. [Read more about roles](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).

## Key Concepts

- **Role**: An atomic permission level within a resource (e.g., "read-only" on a specific AWS account)
- **Resource**: The target system this role provides access to
- **Workflow**: The approval flow triggered when a user requests this role
- **Requestable**: Whether users can submit JIT access requests for this role
- **Allowed Durations**: The time limits for which access can be granted
- **Prerequisite Permissions**: Roles automatically granted alongside this role

## When to Use This Data Source

- Referencing a specific role by ID in a bundle, policy, or prerequisite permission
- Inspecting a role's configuration without managing it via Terraform
- Retrieving a role's associated resource and integration details

## Example Usage

### Look Up a Role by ID

```terraform
data "entitle_role" "prod_admin" {
  id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

output "role_name" {
  value = data.entitle_role.prod_admin.name
}
```

### Reference a Role in a Bundle

```terraform
data "entitle_role" "github_read" {
  id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

resource "entitle_bundle" "dev_bundle" {
  name        = "Developer Bundle"
  description = "Standard developer access"

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  roles = [{
    id = data.entitle_role.github_read.id
  }]

  allowed_durations = [28800, 86400]
}
```

### Use entitle_roles to Find a Role Dynamically

For finding roles by name within a resource, use the `entitle_roles` data source instead:

```terraform
data "entitle_roles" "search" {
  resource_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  filter { search = "admin" }
}

data "entitle_role" "admin_role" {
  id = data.entitle_roles.search.roles[0].id
}
```

## Query Parameters

### Required

- `id` (String) The unique identifier of the role to retrieve (UUID format).

## Returned Attributes

- `id` (String) The role's unique identifier (UUID format).
- `name` (String) The role's display name.
- `requestable` (Boolean) Whether users can submit JIT access requests for this role.
- `allowed_durations` (Set of Number) The available access duration options in seconds (`-1` = permanent).
- `resource` (Attributes) The resource this role belongs to:
    - `id` (String) The resource's unique identifier.
    - `name` (String) The resource's display name.
- `workflow` (Attributes) The approval workflow for this role:
    - `id` (String) The workflow's unique identifier.
    - `name` (String) The workflow's display name.
- `prerequisite_permissions` (Attributes List) Roles automatically granted alongside this role:
    - `default` (Boolean) Whether this prerequisite is automatically included.
    - `role` (Attributes) The prerequisite role's details.
- `virtualized_role` (Attributes) The virtualized role mapping, if any:
    - `id` (String) The virtualized role's unique identifier.
    - `name` (String) The virtualized role's name.

## Finding Role IDs

Use the `entitle_roles` data source to search for roles within a resource:

```terraform
data "entitle_roles" "search" {
  resource_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  filter { search = "read" }
}

output "found_role_ids" {
  value = data.entitle_roles.search.roles[*].id
}
```

Or navigate to **Integrations** → select your integration → **Resources** → select your resource → **Roles** in the Entitle UI, then click the role to find its UUID in the URL.
