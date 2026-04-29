An Entitle Policy is a birthright rule that automatically manages users' permissions based on their group membership. When a user joins the group — for example, upon joining the organization — they are automatically granted the permissions defined in the policy. When they leave the group — for example, upon leaving the organization — those permissions are automatically revoked.

Use this data source to look up an existing policy by ID, for example to reference it in outputs, reference its configuration in other resources, or inspect its current state. [Read more about policies](https://docs.beyondtrust.com/entitle/docs/birthright-policies).

## Key Concepts

- **Policy**: A rule mapping IdP group membership to a set of automatic permissions
- **in_groups**: The IdP groups that trigger the policy
- **roles**: Individual roles automatically granted to group members
- **bundles**: Collections of permissions automatically granted to group members
- **sort_order**: The priority order when multiple policies apply

## When to Use This Data Source

- Referencing an existing policy in Terraform outputs or other resources
- Inspecting policy configuration without managing it via Terraform
- Looking up policy IDs for use in other automation or scripts

## Example Usage

### Look Up a Policy by ID

```terraform
data "entitle_policy" "onboarding" {
  id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

output "policy_id" {
  value = data.entitle_policy.onboarding.id
}
```

## Query Parameters

### Required

- `id` (String) The unique identifier of the policy to retrieve (UUID format).

## Returned Attributes

- `id` (String) The policy's unique identifier (UUID format).
- `sort_order` (Number) The evaluation priority of the policy. Lower numbers are processed first.
- `number` (Number) The sequential policy number assigned by Entitle.
- `in_groups` (Attributes List) The IdP groups or schedules the policy applies to:
    - `id` (String) The group or schedule's unique identifier.
    - `type` (String) `"group"` or `"schedule"`.
    - `name` (String) The group or schedule's display name.
- `roles` (Attributes List) Roles automatically granted by the policy:
    - `id` (String) The role's unique identifier.
    - `name` (String) The role's display name.
    - `resource` (Attributes) The resource and integration associated with the role.
- `bundles` (Attributes List) Bundles automatically granted by the policy:
    - `id` (String) The bundle's unique identifier.
    - `name` (String) The bundle's display name.

## Finding Policy IDs

To find the UUID of a policy:

1. Log in to the Entitle UI
2. Navigate to the **Policies** section
3. Click on the policy
4. The policy ID (UUID) will be visible in the browser URL
    - Example: `https://app.entitle.io/policies/a1b2c3d4-e5f6-7890-abcd-ef1234567890`
