---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "entitle_user Data Source - terraform-provider-entitle"
subcategory: ""
description: |-
  Defines an Entitle User, which represents organization's employee. Read more about users https://docs.beyondtrust.com/entitle/docs/users.
---

# entitle_user (Data Source)

Defines an Entitle User, which represents organization's employee. [Read more about users](https://docs.beyondtrust.com/entitle/docs/users).



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `email` (String) Entitle User email address (identifier)

### Read-Only

- `created_at` (String) Entitle User creation time
- `family_name` (String) Entitle User family name
- `given_name` (String) Entitle User given name
- `id` (String) Entitle User identifier in uuid format
