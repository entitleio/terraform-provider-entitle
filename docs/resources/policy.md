---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "entitle_policy Resource - terraform-provider-entitle"
subcategory: ""
description: |-
  Entitle policy is a rule which manages users birthright permissions automatically, a group of users is entitled to a set of permissions. When a user joins the group, e.g. upon joining the organization, he will be granted with the permissions defined for the group automatically, and upon leaving the group, e.g. leaving the organization, the permissions will be revoked automatically. Read more about policies https://docs.beyondtrust.com/entitle/docs/birthright-policies.
---

# entitle_policy (Resource)

Entitle policy is a rule which manages users birthright permissions automatically, a group of users is entitled to a set of permissions. When a user joins the group, e.g. upon joining the organization, he will be granted with the permissions defined for the group automatically, and upon leaving the group, e.g. leaving the organization, the permissions will be revoked automatically. [Read more about policies](https://docs.beyondtrust.com/entitle/docs/birthright-policies).

## Example Usage

```terraform
resource "entitle_policy" "example" {
  in_groups = [{
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }]

  roles = [{
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `in_groups` (Attributes List) The list of identity provider (IdP) groups that the policy applies to. Users in these groups receive the defined roles or bundles. (see [below for nested schema](#nestedatt--in_groups))

### Optional

- `bundles` (Attributes List) A list of bundles (collections of entitlements) to be assigned by the policy. (see [below for nested schema](#nestedatt--bundles))
- `roles` (Attributes List) A list of roles that the policy assigns to users. Each role grants access to a specific resource. (see [below for nested schema](#nestedatt--roles))

### Read-Only

- `id` (String) Entitle Policy identifier in uuid format

<a id="nestedatt--in_groups"></a>
### Nested Schema for `in_groups`

Required:

- `type` (String) The type of group source ("group" or "schedule").

Optional:

- `id` (String) The unique identifier or email address of the IdP group.

Read-Only:

- `name` (String) The name of the group.


<a id="nestedatt--bundles"></a>
### Nested Schema for `bundles`

Optional:

- `id` (String) The identifier of the bundle to be assigned.

Read-Only:

- `name` (String) The name of the bundle.


<a id="nestedatt--roles"></a>
### Nested Schema for `roles`

Optional:

- `id` (String) The identifier of the role to be granted by the policy.

Read-Only:

- `name` (String) The name of the role.
- `resource` (Attributes) The specific resource associated with the role. (see [below for nested schema](#nestedatt--roles--resource))

<a id="nestedatt--roles--resource"></a>
### Nested Schema for `roles.resource`

Read-Only:

- `id` (String) The unique identifier of the resource.
- `integration` (Attributes) The integration that the resource belongs to. (see [below for nested schema](#nestedatt--roles--resource--integration))
- `name` (String) The display name of the resource.

<a id="nestedatt--roles--resource--integration"></a>
### Nested Schema for `roles.resource.integration`

Read-Only:

- `application` (Attributes) The application that the integration is connected to. (see [below for nested schema](#nestedatt--roles--resource--integration--application))
- `id` (String) The identifier of the integration.
- `name` (String) The display name of the integration.

<a id="nestedatt--roles--resource--integration--application"></a>
### Nested Schema for `roles.resource.integration.application`

Read-Only:

- `name` (String) The name of the connected application.
