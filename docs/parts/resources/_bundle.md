An Entitle Bundle is a cross-application package of permissions that can be requested, approved, or revoked in a single action. Bundles group multiple roles across different applications and resources into a logical unit — think of a bundle as a "super role" that grants access to everything an employee needs for a particular function or project.

Admins create bundles to simplify access requests for end users. Instead of requesting individual roles across five different systems, a user can request the "Junior Accountant" bundle and receive all required permissions at once. Bundles can also be assigned automatically via policies for birthright access. [Read more about bundles](https://docs.beyondtrust.com/entitle/docs/bundles).

## Key Concepts

- **Bundle**: A named collection of roles across one or more applications, requestable as a single unit
- **Roles**: The individual permissions included in the bundle; each role belongs to a specific resource
- **Workflow**: The approval process triggered when a user requests this bundle
- **Allowed Durations**: Time limits for bundle access (can override the organization default)
- **Category**: An organizational label for the bundle (e.g., "Marketing", "Engineering")
- **Tags**: Searchable metadata for discovery and filtering

## When to Use Bundles

- **Onboarding packages**: Create an "Engineer Onboarding" bundle that grants GitHub, AWS dev, and Jira access in one request
- **Project access**: A "Project X" bundle that grants temporary access to all systems needed for a specific initiative
- **Role-based packages**: "Junior Developer" vs "Senior Developer" bundles with different access levels
- **Cross-application access**: When a user's job function requires permissions across multiple integrated systems
- **Policy-based birthright access**: Assign bundles in policies to automatically provision standard access on group membership

## Example Usage

### Basic Bundle

A simple bundle with a single role and workflow:

```terraform
resource "entitle_bundle" "aws_dev_access" {
  name        = "AWS Dev Access"
  description = "Standard read access to the development AWS account"
  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }
  roles = [{
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }]
  allowed_durations = [3600, 28800]
}
```

### Multi-Application Bundle

Grant access across multiple applications in one request:

```terraform
resource "entitle_bundle" "engineer_onboarding" {
  name        = "Engineer Onboarding"
  description = "Standard access package for new engineers — GitHub, AWS dev read, and Jira project membership"
  category    = "Engineering"
  tags        = ["onboarding", "engineering", "standard"]

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  roles = [
    { id = "7d080bfa-9143-11ee-b9d1-0242ac120002" },  # GitHub read
    { id = "7d080bfa-9143-11ee-b9d1-0242ac120003" },  # AWS dev read
    { id = "7d080bfa-9143-11ee-b9d1-0242ac120004" },  # Jira project member
    { id = "7d080bfa-9143-11ee-b9d1-0242ac120005" }   # Slack engineering channel
  ]

  allowed_durations = [-1]  # Permanent access (for onboarding)
}
```

### Project-Specific Bundle

Temporary access package for a specific project:

```terraform
resource "entitle_bundle" "project_alpha" {
  name        = "Project Alpha Access"
  description = "Temporary access to all systems required for Project Alpha"
  category    = "Projects"
  tags        = ["project-alpha", "temporary", "cross-functional"]

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  roles = [
    { id = "7d080bfa-9143-11ee-b9d1-0242ac120002" },  # Alpha DB read
    { id = "7d080bfa-9143-11ee-b9d1-0242ac120003" },  # Alpha S3 bucket
    { id = "7d080bfa-9143-11ee-b9d1-0242ac120004" }   # Alpha deployment pipeline
  ]

  # Allow 8h, 24h, or 7-day access windows
  allowed_durations = [28800, 86400, 604800]
}
```

### Bundle with Category and Tags

Organize bundles with categories and searchable tags for easy discovery:

```terraform
resource "entitle_bundle" "finance_tools" {
  name        = "Finance Team Tools"
  description = "Permissions bundle for finance team members — includes accounting, reporting, and analytics access"
  category    = "Finance"
  tags        = ["finance", "accounting", "reporting", "analytics"]

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  roles = [
    { id = "7d080bfa-9143-11ee-b9d1-0242ac120002" },  # QuickBooks admin
    { id = "7d080bfa-9143-11ee-b9d1-0242ac120003" },  # Tableau viewer
    { id = "7d080bfa-9143-11ee-b9d1-0242ac120004" }   # Finance S3 read
  ]

  allowed_durations = [-1]
}
```

### Bundle Used in a Policy (Birthright Access)

Bundles integrate directly with policies for automatic provisioning:

```terraform
resource "entitle_bundle" "standard_dev_bundle" {
  name        = "Standard Developer Bundle"
  description = "Base developer access — automatically assigned to all engineers"
  category    = "Engineering"

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  roles = [
    { id = "7d080bfa-9143-11ee-b9d1-0242ac120002" },
    { id = "7d080bfa-9143-11ee-b9d1-0242ac120003" }
  ]

  allowed_durations = [-1]
}

# Assign the bundle via a birthright policy
resource "entitle_policy" "all_engineers" {
  in_groups = [{
    id   = "7d080bfa-9143-11ee-b9d1-0242ac120004"
    type = "group"
  }]

  bundles = [{
    id = entitle_bundle.standard_dev_bundle.id
  }]
}
```

### Dynamic Bundle Using Data Sources

Compose a bundle using roles discovered dynamically:

```terraform
data "entitle_integration" "github" {
  name = "GitHub Production"
}

data "entitle_resources" "github_repos" {
  integration_id = data.entitle_integration.github.id
  filter { search = "backend" }
}

data "entitle_roles" "github_read" {
  resource_id = data.entitle_resources.github_repos.resources[0].id
  filter { search = "read" }
}

data "entitle_workflow" "auto_approve" {
  name = "Auto-Approve"
}

resource "entitle_bundle" "dynamic_bundle" {
  name        = "Backend GitHub Read Bundle"
  description = "Read access to backend GitHub repositories"
  category    = "Engineering"

  workflow = {
    id = data.entitle_workflow.auto_approve.id
  }

  roles = [{
    id = data.entitle_roles.github_read.roles[0].id
  }]

  allowed_durations = [28800, 86400]
}
```

## Attributes Reference

### Required

- `name` (String) The display name of the bundle. This is what users see in the access request catalog. Length must be between 2 and 50 characters.
- `description` (String) An extended description of the bundle explaining what access it grants and who it's for. For example: `"Permissions bundle for junior accountants"`.
- `workflow` (Attributes) The approval workflow triggered when a user requests this bundle. See [workflow](#workflow-attribute) below.
- `roles` (Attributes List) The roles included in this bundle. See [roles](#roles-attribute) below.
- `allowed_durations` (Set of Number) The access duration options (in seconds) available when requesting this bundle. Use `-1` for permanent access. Overrides the organization default. Common values:
    - `3600` = 1 hour
    - `28800` = 8 hours
    - `86400` = 24 hours (1 day)
    - `604800` = 7 days
    - `-1` = permanent

### Optional

- `category` (String) An organizational category for this bundle. Typically describes a department or team (e.g., `"Marketing"`, `"Engineering"`, `"Finance"`). Can be a new value to create a new category.
- `tags` (Set of String) Searchable metadata tags for this bundle (e.g., `"accounting"`, `"ATL_Marketing"`, `"Production_Line_14"`). Helps users find the bundle in the request catalog.

### Read-Only

- `id` (String) The unique identifier of the bundle (UUID format).

### workflow attribute

- `id` (Required, String) The unique identifier of the workflow for this bundle. Obtain from the `entitle_workflow` data source.
- `name` (Read-Only, String) The name of the workflow.

### roles attribute

Each role entry in the bundle:

- `id` (Optional, String) The unique identifier of the role to include. Obtain role IDs from the `entitle_roles` data source.
- `name` (Read-Only, String) The name of the role.
- `resource` (Read-Only, Attributes) The resource and integration associated with this role.

## Import

Existing bundles can be imported using their UUID:

```shell
terraform import entitle_bundle.example a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

### Finding the Bundle ID

To find the UUID of an existing bundle:

1. Log in to the Entitle UI
2. Navigate to the **Bundles** section
3. Click on the bundle you want to import
4. The bundle ID (UUID) will be visible in the browser URL
    - Example: `https://app.entitle.io/bundles/a1b2c3d4-e5f6-7890-abcd-ef1234567890`

Alternatively, use the `entitle_bundle` data source to look up a bundle by name:

```terraform
data "entitle_bundle" "existing" {
  name = "My Existing Bundle"
}

output "bundle_id" {
  value = data.entitle_bundle.existing.id
}
```

## Notes and Best Practices

### Bundle Design

- Keep bundles focused on a specific job function or project — avoid "kitchen sink" bundles with excessive permissions
- Name bundles clearly so end users understand what they're requesting (e.g., `"Junior Accountant Tools"` not `"Bundle A"`)
- Use `description` to explain the business purpose and who the bundle is intended for
- Use `tags` generously to improve discoverability in the request catalog

### Allowed Durations

- For onboarding bundles (birthright use case), include `-1` for permanent access
- For project-specific or elevated access bundles, use explicit time windows to enforce least-privilege
- Match duration options to your organization's access policies and audit requirements

### Bundles vs Individual Roles

- Bundles are preferred when a user needs multiple permissions across applications
- Use individual role requests when access is to a single system with no related dependencies
- Bundles reduce friction for end users and make access requests easier to understand and approve

### Workflows and Bundles

- The bundle's workflow applies to the entire bundle request — all roles in the bundle are granted or denied together
- If different roles in the bundle have very different risk profiles, consider splitting into separate bundles with different workflows
