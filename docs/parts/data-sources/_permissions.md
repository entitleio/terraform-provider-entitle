Retrieve a list of Entitle Permissions — active, granted entitlements representing users who currently hold access to specific roles within resources and integrations. Permissions are the result of access being granted, either through an approved JIT request or a birthright policy.

Use this data source to discover existing permissions for auditing, to find permission IDs for import into the `entitle_permission` resource, or to analyze current access states across your integrations.

## Key Concepts

- **Permission**: An active grant of a specific role to a specific user (actor)
- **Actor**: The user who holds the permission
- **Role**: The specific access level that was granted
- **Filter**: All filter parameters are optional — an empty filter returns all permissions
- **Difference from entitle_permission**: Use `entitle_permissions` (plural, data source) to query and discover; use `entitle_permission` (singular, resource) to manage individual permissions via Terraform

## When to Use This Data Source

- **Auditing**: List all active permissions for a role, account, resource, or integration
- **Pre-import discovery**: Find permission IDs to import into `entitle_permission` resources for Terraform management
- **Access reporting**: Generate outputs showing who has access to what
- **Compliance checks**: Verify the current permission state against expected configurations

## Example Usage

### Get All Permissions (No Filter)

Retrieve all permissions across the organization:

```terraform
data "entitle_permissions" "all" {}

output "total_permissions" {
  value = length(data.entitle_permissions.all.permissions)
}
```

### Filter by Role

Get all users who currently have a specific role:

```terraform
data "entitle_permissions" "role_holders" {
  filter {
    role_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }
}

output "users_with_role" {
  value = data.entitle_permissions.role_holders.permissions[*].actor.email
}
```

### Filter by Account

Get all permissions held by a specific user account:

```terraform
data "entitle_permissions" "user_permissions" {
  filter {
    account_id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }
}

output "user_access" {
  value = [for p in data.entitle_permissions.user_permissions.permissions : {
    role_name        = p.role.name
    resource_name    = p.role.resource.name
    integration_name = p.role.resource.integration.name
  }]
}
```

### Filter by Integration

Get all permissions under a specific integration:

```terraform
data "entitle_permissions" "github_permissions" {
  filter {
    integration_id = "7d080bfa-9143-11ee-b9d1-0242ac120003"
  }
}

output "github_access_count" {
  value = length(data.entitle_permissions.github_permissions.permissions)
}
```

### Filter by Resource

Get all permissions for a specific resource:

```terraform
data "entitle_permissions" "prod_db_permissions" {
  filter {
    resource_id = "7d080bfa-9143-11ee-b9d1-0242ac120004"
  }
}

output "prod_db_access" {
  value = data.entitle_permissions.prod_db_permissions.permissions
}
```

### Import All Role Permissions into Terraform Management

Discover permissions and bring them under Terraform management:

```terraform
data "entitle_permissions" "managed_permissions" {
  filter {
    role_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }
}

resource "entitle_permission" "managed" {
  for_each = {
    for p in data.entitle_permissions.managed_permissions.permissions :
    p.id => p
  }

  id = each.value.id
}
```

### Text Search

Search permissions using a free-text query:

```terraform
data "entitle_permissions" "search" {
  filter {
    search = "admin"
  }
}
```

## Query Parameters

### Optional

- `filter` (Block) All filter fields are optional:
    - `role_id` (String) Filter by role ID (UUID) — returns all permissions for this role.
    - `account_id` (String) Filter by account/user ID (UUID) — returns all permissions held by this user.
    - `integration_id` (String) Filter by integration ID (UUID) — returns all permissions under this integration.
    - `resource_id` (String) Filter by resource ID (UUID) — returns all permissions for this resource.
    - `search` (String) Free-text search across permissions.

## Returned Attributes

- `permissions` (Attributes List) The list of permissions matching the query:
    - `id` (String) The permission's unique identifier.
    - `path` (String) The hierarchical path describing where the permission was granted.
    - `types` (Set of String) The categories of permission granted.
    - `actor` (Attributes) The user who holds the permission:
        - `id` (String) The actor's unique identifier.
        - `email` (String) The actor's email address.
    - `role` (Attributes) The granted role:
        - `id` (String) The role's unique identifier.
        - `name` (String) The role's display name.
        - `resource` (Attributes) The resource associated with the role:
            - `id` (String) The resource's unique identifier.
            - `name` (String) The resource's display name.
            - `integration` (Attributes):
                - `id` (String) The integration's unique identifier.
                - `name` (String) The integration's display name.
                - `application` (Attributes):
                    - `name` (String) The application's name.

## Notes

- An empty filter block (or omitting `filter` entirely) returns all permissions — use with caution in large organizations
- Combine filters for more specific results (e.g., `role_id` + `search`)
- Use the `for_each` pattern with the `entitle_permission` resource to bulk-import discovered permissions into Terraform state
