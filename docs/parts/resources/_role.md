An Entitle Role defines a specific permission that can be requested and granted within a resource. Roles represent the most granular unit of access in Entitle — they map to a real permission level inside an integrated application (e.g., "read-only" or "admin" within a specific AWS account, MongoDB database, or GitHub repository).

Each role belongs to exactly one resource, and can be linked to a workflow to define the approval process for access requests. Roles can be grouped into bundles for cross-application access packages, or assigned directly through policies for birthright access. [Read more about roles](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).

## Key Concepts

- **Role**: The atomic permission unit — a named access level within a specific resource
- **Resource**: The target system the role provides access to (e.g., a specific AWS account)
- **Workflow**: The approval flow triggered when a user requests this role
- **Allowed Durations**: The time limits for which access can be granted (overrides integration/workflow defaults)
- **Requestable**: Whether users can submit JIT access requests for this role
- **Prerequisite Permissions**: Roles automatically granted alongside this role when access is approved
- **Virtualized Role**: An abstract role that maps to different real roles depending on the resource

## When to Use Roles

- Define granular access levels within a resource (e.g., `readonly`, `contributor`, `admin`)
- Attach different approval workflows to different access levels (e.g., read is auto-approved, write requires manager)
- Control the maximum duration for which a specific permission level can be held
- Set up prerequisite permissions that must always accompany a role (e.g., read access must come with a viewer role)

## Example Usage

### Basic Role

Create a requestable role with a workflow and no duration override:

```terraform
resource "entitle_role" "dev_read" {
  name      = "Developer Read"
  requestable = true

  resource = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  allowed_durations = [-1]  # -1 means use the workflow/integration default
}
```

### Role with Specific Allowed Durations

Restrict access to specific time windows only (in seconds):

```terraform
resource "entitle_role" "prod_admin" {
  name        = "Production Admin"
  requestable = true

  resource = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  # Allow only 1h, 4h, or 8h access windows
  allowed_durations = [3600, 14400, 28800]
}
```

### Non-Requestable Role (Policy-Only Access)

A role that can only be assigned via policies (birthright), not requested on-demand:

```terraform
resource "entitle_role" "base_access" {
  name        = "Base Employee Access"
  requestable = false

  resource = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  allowed_durations = [-1]
}
```

### Role with Prerequisite Permissions

Automatically grant a companion role whenever this role is approved. Useful when one permission logically requires another (e.g., "write access" should always include "read access"):

```terraform
resource "entitle_role" "db_write" {
  name        = "Database Write"
  requestable = true

  resource = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  allowed_durations = [3600, 7200]

  prerequisite_permissions = [
    {
      default = true  # Automatically granted as part of the request
      role = {
        id = "7d080bfa-9143-11ee-b9d1-0242ac120003"  # db_read role
      }
    }
  ]
}
```

### Role with Virtualized Role

Map a logical role to a virtualized role (used for dynamic permission mapping):

```terraform
resource "entitle_role" "virtualized_access" {
  name        = "Virtualized Contributor"
  requestable = true

  resource = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  allowed_durations = [-1]

  virtualized_role = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120003"
  }
}
```

### Role with Multiple Allowed Durations Including Permanent

Include `-1` to allow permanent access alongside timed windows:

```terraform
resource "entitle_role" "flexible_access" {
  name        = "Flexible Access"
  requestable = true

  resource = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  # Allow 1h, 8h, or permanent access
  allowed_durations = [3600, 28800, -1]
}
```

### Full Example: Tiered Access Roles for a Resource

Define a full set of roles for a single resource with escalating approval requirements:

```terraform
data "entitle_workflow" "auto_approve" {
  name = "Auto-Approve"
}

data "entitle_workflow" "manager_approval" {
  name = "Manager Approval"
}

data "entitle_workflow" "security_approval" {
  name = "Security Approval"
}

# Read-only: auto-approved, short durations
resource "entitle_role" "app_read" {
  name        = "App Read"
  requestable = true

  resource = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = data.entitle_workflow.auto_approve.id
  }

  allowed_durations = [3600, 14400, 28800]
}

# Write: requires manager, longer durations
resource "entitle_role" "app_write" {
  name        = "App Write"
  requestable = true

  resource = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = data.entitle_workflow.manager_approval.id
  }

  allowed_durations = [3600, 28800]

  prerequisite_permissions = [{
    default = true
    role = {
      id = entitle_role.app_read.id
    }
  }]
}

# Admin: requires security team, short durations only
resource "entitle_role" "app_admin" {
  name        = "App Admin"
  requestable = true

  resource = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = data.entitle_workflow.security_approval.id
  }

  allowed_durations = [3600, 7200]
}
```

## Attributes Reference

### Required

- `name` (String) The display name for the role. This is what users see when requesting access. Length must be between 2 and 50 characters.
- `requestable` (Boolean) Whether users can submit JIT access requests for this role. Set to `false` for roles that should only be assigned via policies (birthright access).
- `resource` (Attributes) The resource this role belongs to. See [resource](#resource-attribute) below.
- `allowed_durations` (Set of Number) The access durations (in seconds) available for this role. Use `-1` for permanent/unlimited access. Common values:
    - `3600` = 1 hour
    - `7200` = 2 hours
    - `14400` = 4 hours
    - `28800` = 8 hours
    - `86400` = 24 hours
    - `-1` = permanent (no expiry)

### Optional

- `workflow` (Attributes) The approval workflow for JIT access requests to this role. If not set, the resource or integration workflow is used as a fallback. See [workflow](#workflow-attribute) below.
- `prerequisite_permissions` (Attributes List) Roles that are automatically granted alongside this role when a request is approved. See [prerequisite_permissions](#prerequisite_permissions) below.
- `virtualized_role` (Attributes) A virtualized role mapping for dynamic permission assignment. See [virtualized_role](#virtualized_role) below.

### Read-Only

- `id` (String) The unique identifier of the role (UUID format).

### resource attribute

- `id` (Required, String) The unique identifier of the resource this role belongs to. Obtain resource IDs from the `entitle_resource` data source or from `entitle_resources`.
- `name` (Read-Only, String) The name of the resource.

### workflow attribute

- `id` (Required, String) The unique identifier of the workflow to use for JIT access requests to this role. Obtain workflow IDs from the `entitle_workflow` data source.
- `name` (Read-Only, String) The name of the assigned workflow.

### prerequisite_permissions

Each prerequisite permission entry:

- `role` (Required, Attributes) The role to automatically grant alongside this one:
    - `id` (Required, String) The unique identifier of the prerequisite role.
    - `name` (Read-Only, String) The name of the prerequisite role.
    - `resource` (Read-Only, Attributes) The resource associated with the prerequisite role.
- `default` (Optional, Boolean) When `true`, this prerequisite permission is automatically included with the role request without requiring separate user selection. Defaults to `false`.

### virtualized_role

- `id` (Required, String) The unique identifier of the virtualized role to map to.
- `name` (Read-Only, String) The name of the virtualized role.

## Import

Existing roles can be imported using their UUID:

```shell
terraform import entitle_role.example a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

### Finding the Role ID

To find the UUID of an existing role:

1. Log in to the Entitle UI
2. Navigate to **Integrations** → select your integration → **Resources** → select your resource → **Roles**
3. Click on the role you want to import
4. The role ID (UUID) will be visible in the browser URL
    - Example: `https://app.entitle.io/roles/a1b2c3d4-e5f6-7890-abcd-ef1234567890`

Alternatively, use the `entitle_roles` data source to list roles for a resource and find the ID programmatically:

```terraform
data "entitle_roles" "my_roles" {
  resource_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  filter { search = "admin" }
}

output "role_ids" {
  value = data.entitle_roles.my_roles.roles[*].id
}
```

## Notes and Best Practices

### Allowed Durations

- `-1` means "use the organization default" or "permanent" depending on configuration — verify with your Entitle admin what this maps to in your org
- Providing multiple values gives users a choice at request time
- Keep high-privilege roles to short durations only (e.g., production admin should not allow 24h+ access)
- Duration constraints here override the workflow's `under_duration` rules

### Requestable vs Non-Requestable

- Set `requestable = true` for JIT (just-in-time) roles that users can request on-demand
- Set `requestable = false` for roles that are only granted automatically via policies (birthright access) and should never appear in the self-service request catalog

### Prerequisite Permissions

- Use prerequisite permissions to model access hierarchies (e.g., write access always includes read)
- Set `default = true` to make the prerequisite automatic — the user doesn't need to select it separately
- Avoid circular prerequisite dependencies

### Workflow Assignment

- If a role has no workflow, Entitle falls back to the parent resource's workflow, then the integration's workflow
- Assign role-level workflows when you need different approval chains for different access levels within the same resource
