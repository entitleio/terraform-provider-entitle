An Entitle Resource represents a specific asset or system within an integration that can be accessed and governed through Entitle. Resources are the middle tier of the access model (Integration → Resource → Role) — they group related roles together and provide shared ownership, workflow, and policy configuration.

Use this data source to look up a specific resource by ID, for example to retrieve its configuration, find its associated integration, or reference it when creating roles. [Read more about resources](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).

## Key Concepts

- **Resource**: A named, governable asset within an integration (e.g., a specific AWS account, GitHub repo, database)
- **Integration**: The parent application connection this resource belongs to
- **Owner**: The user responsible for this resource
- **Workflow**: The default approval process for JIT access requests to roles on this resource
- **Requestable**: Whether users can request access to roles on this resource

## When to Use This Data Source

- Retrieving a resource's ID or details when creating roles (`entitle_role`)
- Inspecting a resource's current owner, workflow, and access configuration
- Looking up resource metadata (tags, description) for auditing

## Example Usage

### Look Up a Resource by ID

```terraform
data "entitle_resource" "aws_dev" {
  id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

output "resource_name" {
  value = data.entitle_resource.aws_dev.name
}
```

### Use Resource ID When Creating a Role

```terraform
data "entitle_resource" "github_repo" {
  id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

resource "entitle_role" "repo_read" {
  name        = "Repository Read"
  requestable = true

  resource = {
    id = data.entitle_resource.github_repo.id
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  allowed_durations = [3600, 28800]
}
```

### Inspect Resource Configuration

```terraform
data "entitle_resource" "prod_db" {
  id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

output "resource_details" {
  value = {
    name              = data.entitle_resource.prod_db.name
    requestable       = data.entitle_resource.prod_db.requestable
    allowed_durations = data.entitle_resource.prod_db.allowed_durations
    owner_email       = data.entitle_resource.prod_db.owner.email
    integration_name  = data.entitle_resource.prod_db.integration.name
    tags              = data.entitle_resource.prod_db.tags
  }
}
```

### Find a Resource Using entitle_resources First

When you don't know the resource ID, use the `entitle_resources` data source to search first:

```terraform
data "entitle_integration" "aws" {
  name = "AWS Production"
}

data "entitle_resources" "search" {
  integration_id = data.entitle_integration.aws.id
  filter { search = "dev-account" }
}

data "entitle_resource" "dev_account" {
  id = data.entitle_resources.search.resources[0].id
}
```

## Query Parameters

### Required

- `id` (String) The unique identifier of the resource to retrieve (UUID format).

## Returned Attributes

- `id` (String) The resource's unique identifier (UUID format).
- `name` (String) The resource's display name.
- `description` (String) The resource's system-generated description.
- `user_defined_description` (String) The custom description provided by the admin.
- `requestable` (Boolean) Whether users can request access to roles on this resource.
- `allowed_durations` (Set of Number) Available access duration options in seconds.
- `tags` (Set of String) System-managed metadata tags.
- `user_defined_tags` (Set of String) Admin-defined metadata tags.
- `integration` (Attributes) The parent integration:
    - `id` (String) The integration's unique identifier.
    - `name` (String) The integration's display name.
    - `application` (Attributes):
        - `name` (String) The application type name.
- `owner` (Attributes) The resource's primary owner:
    - `id` (String) The owner's unique identifier.
    - `email` (String) The owner's email address.
- `workflow` (Attributes) The default approval workflow:
    - `id` (String) The workflow's unique identifier.
    - `name` (String) The workflow's display name.
- `maintainers` (Attributes List) Secondary owners:
    - `type` (String) `"user"` or `"group"`.
    - `entity` (Attributes):
        - `id` (String) The maintainer's unique identifier.
        - `email` (String) The maintainer's email address.
- `prerequisite_permissions` (Attributes List) Roles auto-granted alongside roles on this resource.
