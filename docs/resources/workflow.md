---
page_title: "entitle_workflow Resource - terraform-provider-entitle"
subcategory: ""
description: |-
  Manages an Entitle workflow that defines approval processes for just-in-time access requests.
---

# entitle_workflow (Resource)

A workflow defines the approval process for just-in-time access requests in Entitle. Workflows specify who must approve access requests, in what order, for how long, and under what conditions.

## Key Concepts

- **Workflow**: A set of rules that determine the approval process
- **Rules**: Conditional approval requirements based on duration, schedule, or groups
- **Approval Steps**: Sequential stages of approval (e.g., manager then security team)
- **Approval Entities**: Who can approve at each step (users, groups, managers, automatic)

## When to Use Workflows

Workflows should be created for different scenarios such as:

- Different approval requirements for different access durations (short vs long)
- Business hours vs after-hours access
- Different approval chains for different teams/groups
- Emergency/break-glass access with automatic approval
- High-privilege access requiring multiple approvals

## Example Usage

### Basic Auto-Approval Workflow

Simple workflow that automatically approves all requests:

```terraform
resource "entitle_workflow" "auto_approve" {
  name        = "Auto-Approve Development"
  description = "Automatically approve development environment access"
  
  rules = [{
    sort_order     = 1
    under_duration = 14400  # 4 hours
    any_schedule   = true
    
    approval_flow = {
      steps = [{
        sort_order = 1
        operator   = "or"
        
        approval_entities = [{
          type = "Automatic"
        }]
        
        notified_entities = []
      }]
    }
    
    in_groups    = []
    in_schedules = []
  }]
}
```

### Manager Approval Workflow

Requires the requester's direct manager to approve:

```terraform
resource "entitle_workflow" "manager_approval" {
  name        = "Manager Approval Required"
  description = "Requests must be approved by the requester's manager"
  
  rules = [{
    sort_order     = 1
    under_duration = 28800  # 8 hours
    any_schedule   = true
    
    approval_flow = {
      steps = [{
        sort_order = 1
        operator   = "or"
        
        approval_entities = [{
          type = "Manager"
        }]
        
        notified_entities = []
      }]
    }
    
    in_groups    = []
    in_schedules = []
  }]
}
```

### Multi-Step Approval Workflow (Sequential)

Requires sequential approvals from manager, then security team:

```terraform
data "entitle_group" "security_team" {
  name = "Security Team"
}

resource "entitle_workflow" "production_access" {
  name        = "Production Access - Multi-Step"
  description = "Manager approval followed by security team approval"
  
  rules = [{
    sort_order     = 1
    under_duration = 7200  # 2 hours
    any_schedule   = true
    
    approval_flow = {
      steps = [
        {
          # First approval: Manager
          sort_order = 1
          operator   = "or"
          
          approval_entities = [{
            type = "Manager"
          }]
          
          notified_entities = []
        },
        {
          # Second approval: Security team
          sort_order = 2
          operator   = "or"
          
          approval_entities = [{
            type = "Group"
            id   = data.entitle_group.security_team.id
          }]
          
          notified_entities = []
        }
      ]
    }
    
    in_groups    = []
    in_schedules = []
  }]
}
```

### Multiple Approval Options (OR Logic)

Any member of security team OR any DevOps engineer can approve:

```terraform
data "entitle_group" "security" {
  name = "Security"
}

data "entitle_group" "devops" {
  name = "DevOps"
}

resource "entitle_workflow" "flexible_approval" {
  name = "Security or DevOps Approval"
  
  rules = [{
    sort_order     = 1
    under_duration = 3600  # 1 hour
    any_schedule   = true
    
    approval_flow = {
      steps = [{
        sort_order = 1
        operator   = "or"  # Any ONE of these approvers is sufficient
        
        approval_entities = [
          {
            type = "Group"
            id   = data.entitle_group.security.id
          },
          {
            type = "Group"
            id   = data.entitle_group.devops.id
          }
        ]
        
        notified_entities = []
      }]
    }
    
    in_groups    = []
    in_schedules = []
  }]
}
```

### Multiple Required Approvals Using AND Operator (Parallel)

Both security team AND compliance team must approve (parallel approvals):

```terraform
data "entitle_group" "security" {
  name = "Security"
}

data "entitle_group" "compliance" {
  name = "Compliance"
}

resource "entitle_workflow" "dual_approval_parallel" {
  name        = "Security AND Compliance Approval (Parallel)"
  description = "Both teams must approve, but can happen in any order"
  
  rules = [{
    sort_order     = 1
    under_duration = 3600
    any_schedule   = true
    
    approval_flow = {
      steps = [{
        sort_order = 1
        operator   = "and"  # ALL entities in this step must approve
        
        approval_entities = [
          {
            type = "Group"
            id   = data.entitle_group.security.id
          },
          {
            type = "Group"
            id   = data.entitle_group.compliance.id
          }
        ]
        
        notified_entities = []
      }]
    }
    
    in_groups    = []
    in_schedules = []
  }]
}
```

### Multiple Required Approvals Using Sequential Steps

Both security team AND compliance team must approve (sequential approvals):

```terraform
data "entitle_group" "security" {
  name = "Security"
}

data "entitle_group" "compliance" {
  name = "Compliance"
}

resource "entitle_workflow" "dual_approval_sequential" {
  name        = "Security THEN Compliance Approval (Sequential)"
  description = "Security must approve first, then compliance"
  
  rules = [{
    sort_order     = 1
    under_duration = 3600
    any_schedule   = true
    
    approval_flow = {
      steps = [
        {
          sort_order = 1
          operator   = "or"
          
          approval_entities = [{
            type = "Group"
            id   = data.entitle_group.security.id
          }]
          
          notified_entities = []
        },
        {
          sort_order = 2
          operator   = "or"
          
          approval_entities = [{
            type = "Group"
            id   = data.entitle_group.compliance.id
          }]
          
          notified_entities = []
        }
      ]
    }
    
    in_groups    = []
    in_schedules = []
  }]
}
```

### Complex Approval: Multiple Steps with AND/OR Logic

Manager approval, then (Security AND Compliance in parallel):

```terraform
data "entitle_group" "security" {
  name = "Security"
}

data "entitle_group" "compliance" {
  name = "Compliance"
}

resource "entitle_workflow" "complex_approval" {
  name        = "Manager, then Security AND Compliance"
  description = "Sequential manager approval followed by parallel dual approval"
  
  rules = [{
    sort_order     = 1
    under_duration = 7200
    any_schedule   = true
    
    approval_flow = {
      steps = [
        {
          # Step 1: Manager must approve first
          sort_order = 1
          operator   = "or"
          
          approval_entities = [{
            type = "Manager"
          }]
          
          notified_entities = []
        },
        {
          # Step 2: Both Security AND Compliance must approve (parallel)
          sort_order = 2
          operator   = "and"
          
          approval_entities = [
            {
              type = "Group"
              id   = data.entitle_group.security.id
            },
            {
              type = "Group"
              id   = data.entitle_group.compliance.id
            }
          ]
          
          notified_entities = []
        }
      ]
    }
    
    in_groups    = []
    in_schedules = []
  }]
}
```

### Duration-Based Rules

Different approval requirements based on requested access duration:

```terraform
data "entitle_group" "security" {
  name = "Security Team"
}

resource "entitle_workflow" "duration_based" {
  name        = "Duration-Based Approval"
  description = "Short access auto-approved, long access requires security"
  
  rules = [
    {
      # Rule 1: Short duration (up to 1 hour) - auto approve
      sort_order     = 1
      under_duration = 3600  # 1 hour
      any_schedule   = true
      
      approval_flow = {
        steps = [{
          sort_order = 1
          operator   = "or"
          
          approval_entities = [{
            type = "Automatic"
          }]
          
          notified_entities = []
        }]
      }
      
      in_groups    = []
      in_schedules = []
    },
    {
      # Rule 2: Longer duration (1-8 hours) - requires manager
      sort_order     = 2
      under_duration = 28800  # 8 hours
      any_schedule   = true
      
      approval_flow = {
        steps = [{
          sort_order = 1
          operator   = "or"
          
          approval_entities = [{
            type = "Manager"
          }]
          
          notified_entities = []
        }]
      }
      
      in_groups    = []
      in_schedules = []
    },
    {
      # Rule 3: Very long duration (over 8 hours) - requires security
      sort_order     = 3
      under_duration = 86400  # 24 hours (catches everything above 8 hours)
      any_schedule   = true
      
      approval_flow = {
        steps = [{
          sort_order = 1
          operator   = "or"
          
          approval_entities = [{
            type = "Group"
            id   = data.entitle_group.security.id
          }]
          
          notified_entities = []
        }]
      }
      
      in_groups    = []
      in_schedules = []
    }
  ]
}
```

### Schedule-Based Rules

Different approval requirements for business hours vs after-hours:

```terraform
data "entitle_schedule" "business_hours" {
  name = "Business Hours (9am-5pm)"
}

data "entitle_group" "security" {
  name = "Security"
}

resource "entitle_workflow" "schedule_based" {
  name = "Business Hours vs After Hours"
  
  rules = [
    {
      # Rule 1: During business hours - auto approve short access
      sort_order     = 1
      under_duration = 14400  # 4 hours
      any_schedule   = false
      
      in_schedules = [data.entitle_schedule.business_hours.id]
      
      approval_flow = {
        steps = [{
          sort_order = 1
          operator   = "or"
          
          approval_entities = [{
            type = "Automatic"
          }]
          
          notified_entities = []
        }]
      }
      
      in_groups = []
    },
    {
      # Rule 2: After hours - requires security approval
      sort_order     = 2
      under_duration = 7200  # 2 hours
      any_schedule   = true  # Applies when business hours schedule doesn't match
      
      approval_flow = {
        steps = [{
          sort_order = 1
          operator   = "or"
          
          approval_entities = [{
            type = "Group"
            id   = data.entitle_group.security.id
          }]
          
          notified_entities = []
        }]
      }
      
      in_groups    = []
      in_schedules = []
    }
  ]
}
```

### Group-Based Rules

Different approval requirements for different teams:

```terraform
data "entitle_group" "developers" {
  name = "Developers"
}

data "entitle_group" "contractors" {
  name = "Contractors"
}

resource "entitle_workflow" "group_based" {
  name = "Different Rules per Group"
  
  rules = [
    {
      # Rule 1: Developers get auto-approval
      sort_order     = 1
      under_duration = 7200  # 2 hours
      any_schedule   = true
      
      in_groups = [data.entitle_group.developers.id]
      
      approval_flow = {
        steps = [{
          sort_order = 1
          operator   = "or"
          
          approval_entities = [{
            type = "Automatic"
          }]
          
          notified_entities = []
        }]
      }
      
      in_schedules = []
    },
    {
      # Rule 2: Contractors need manager approval
      sort_order     = 2
      under_duration = 3600  # 1 hour
      any_schedule   = true
      
      in_groups = [data.entitle_group.contractors.id]
      
      approval_flow = {
        steps = [{
          sort_order = 1
          operator   = "or"
          
          approval_entities = [{
            type = "Manager"
          }]
          
          notified_entities = []
        }]
      }
      
      in_schedules = []
    }
  ]
}
```

### Resource Owner Approval

Requires the owner of the resource being accessed to approve:

```terraform
resource "entitle_workflow" "resource_owner" {
  name = "Resource Owner Approval"
  
  rules = [{
    sort_order     = 1
    under_duration = 7200  # 2 hours
    any_schedule   = true
    
    approval_flow = {
      steps = [{
        sort_order = 1
        operator   = "or"
        
        approval_entities = [{
          type = "ResourceOwner"
        }]
        
        notified_entities = []
      }]
    }
    
    in_groups    = []
    in_schedules = []
  }]
}
```

### Specific User Approval

Requires a specific named user to approve:

```terraform
data "entitle_user" "security_lead" {
  email = "[email protected]"
}

resource "entitle_workflow" "specific_user" {
  name = "Security Lead Approval"
  
  rules = [{
    sort_order     = 1
    under_duration = 3600  # 1 hour
    any_schedule   = true
    
    approval_flow = {
      steps = [{
        sort_order = 1
        operator   = "or"
        
        approval_entities = [{
          type = "User"
          id   = data.entitle_user.security_lead.id
        }]
        
        notified_entities = []
      }]
    }
    
    in_groups    = []
    in_schedules = []
  }]
}
```

### Workflow with Notifications

Notify additional people when requests are made (without requiring their approval):

```terraform
data "entitle_user" "ciso" {
  email = "[email protected]"
}

data "entitle_group" "security_team" {
  name = "Security Team"
}

resource "entitle_workflow" "with_notifications" {
  name = "Manager Approval with CISO Notification"
  
  rules = [{
    sort_order     = 1
    under_duration = 7200
    any_schedule   = true
    
    approval_flow = {
      steps = [{
        sort_order = 1
        operator   = "or"
        
        approval_entities = [{
          type = "Manager"
        }]
        
        # These people will be notified but don't need to approve
        notified_entities = [
          {
            type = "User"
            id   = data.entitle_user.ciso.id
          },
          {
            type = "Group"
            id   = data.entitle_group.security_team.id
          }
        ]
      }]
    }
    
    in_groups    = []
    in_schedules = []
  }]
}
```

### Combining Manager with Fallback Approvers

Provides alternative approval paths if requester has no manager:

```terraform
data "entitle_group" "team_leads" {
  name = "Team Leads"
}

resource "entitle_workflow" "manager_with_fallback" {
  name = "Manager or Team Lead Approval"
  description = "Manager approves, or Team Lead if no manager assigned"
  
  rules = [{
    sort_order     = 1
    under_duration = 7200
    any_schedule   = true
    
    approval_flow = {
      steps = [{
        sort_order = 1
        operator   = "or"
        
        approval_entities = [
          {
            type = "Manager"
          },
          {
            type = "Group"
            id   = data.entitle_group.team_leads.id
          }
        ]
        
        notified_entities = []
      }]
    }
    
    in_groups    = []
    in_schedules = []
  }]
}
```

## Schema

### Required

- `name` (String) The name of the workflow. Must be unique within your Entitle organization.
- `rules` (List of Objects) List of approval rules. At least one rule is required. Rules are evaluated in order based on `sort_order` and the first matching rule is applied. See [Rules](#rules) below.

### Optional

- `description` (String) Human-readable description of the workflow's purpose and usage.

### Read-Only

- `id` (String) The unique identifier of the workflow (UUID format).

## Rules

Each rule in the `rules` list represents a conditional approval requirement. Rules are evaluated in order (by `sort_order`), and the **first matching rule** is applied to the access request.

### Rule Matching Logic

A rule matches an access request when **ALL** of the following conditions are true:

1. The requested duration is **less than or equal to** `under_duration`
2. The requester is in one of the groups specified in `in_groups` (or `in_groups` is empty)
3. The request time matches one of the schedules in `in_schedules` (or `any_schedule` is true)

### Rule Attributes

- `sort_order` (Required, Integer) The evaluation order of this rule. Rules with lower numbers are evaluated first. Must be unique within the workflow.
  
- `under_duration` (Required, Integer) Maximum access duration in seconds for which this rule applies. Requests for access durations up to and including this value will match this rule.
  - Example: `3600` = 1 hour, `7200` = 2 hours, `86400` = 24 hours
  - Use a high value (e.g., `999999999`) for a catch-all rule
  
- `any_schedule` (Required, Boolean) If `true`, this rule applies at any time regardless of schedule. If `false`, the rule only applies during the schedules specified in `in_schedules`.
  - **Note**: Set to `true` if not using schedule-based rules
  - Cannot be `true` if `in_schedules` is not empty

- `in_schedules` (Required, List of Strings) List of schedule IDs during which this rule applies. Leave empty (`[]`) if `any_schedule` is `true`.
  - Schedules define time windows (e.g., "Business Hours", "Weekends")
  - Obtain schedule IDs from `entitle_schedule` data source
  - If multiple schedule IDs are provided, the rule applies if the current time matches ANY of the schedules (OR logic)

- `in_groups` (Required, List of Strings) List of group IDs for which this rule applies. If empty (`[]`), the rule applies to all users.
  - Use this to create different approval flows for different teams
  - Obtain group IDs from `entitle_group` data source
  - **Logic**: User must be in at least ONE of the listed groups (OR logic)

- `approval_flow` (Required, Object) Defines the approval process for requests matching this rule. See [Approval Flow](#approval-flow) below.

## Approval Flow

The `approval_flow` object defines the sequence of approval steps required.

### Approval Flow Attributes

- `steps` (Required, List of Objects) Sequential list of approval steps. Requests must be approved at each step in order (by `sort_order`) before access is granted. See [Approval Steps](#approval-steps) below.

## Approval Steps

Each step represents a stage in the approval process. Steps are executed sequentially based on `sort_order`.

### Approval Step Attributes

- `sort_order` (Required, Integer) The order in which this step is executed. Lower numbers execute first. Must be unique within the approval flow.

- `operator` (Required, String) Defines how multiple `approval_entities` within this step are evaluated:
  - `"or"` - **Any ONE** of the approval entities can approve (at least one must approve to proceed)
  - `"and"` - **ALL** of the approval entities must approve (every entity must approve to proceed)
  
  **Examples:**
  
  **Using "or"**: Any member of Security OR DevOps can approve
  ```terraform
  {
    operator = "or"
    approval_entities = [
      { type = "Group", id = security_group_id },
      { type = "Group", id = devops_group_id }
    ]
  }
  ```
  
  **Using "and"**: Both Security AND Compliance must approve
  ```terraform
  {
    operator = "and"
    approval_entities = [
      { type = "Group", id = security_group_id },
      { type = "Group", id = compliance_group_id }
    ]
  }
  ```

- `approval_entities` (Required, List of Objects) List of entities (users, groups, etc.) who can approve at this step. See [Approval Entities](#approval-entities) below.
  - With `operator = "or"`: At least ONE entity must approve
  - With `operator = "and"`: ALL entities must approve

- `notified_entities` (Required, List of Objects) List of entities who will be notified when a request reaches this step, but are not required to approve. Same structure as `approval_entities`. Can be empty (`[]`).

### Understanding Steps vs Operators

There are two ways to require multiple approvals:

**1. Sequential Approvals (Multiple Steps)**

Use when approvals must happen in a specific order:

```terraform
steps = [
  { sort_order = 1, /* Manager approves first */ },
  { sort_order = 2, /* Then Security approves */ }
]
```
- Step 2 only begins after Step 1 is complete
- Creates a waterfall approval process

**2. Parallel Approvals (operator = "and")**

Use when multiple approvers must approve, but order doesn't matter:

```terraform
steps = [{
  operator = "and"
  approval_entities = [
    { type = "Group", id = security_id },
    { type = "Group", id = compliance_id }
  ]
}]
```
- Both Security and Compliance must approve
- They can approve in any order
- Faster than sequential steps

**3. Combining Both**

Complex workflows can use both techniques:

```terraform
steps = [
  {
    # Step 1: Manager must approve first
    sort_order = 1
    operator = "or"
    approval_entities = [{ type = "Manager" }]
  },
  {
    # Step 2: Then Security AND Compliance (parallel)
    sort_order = 2
    operator = "and"
    approval_entities = [
      { type = "Group", id = security_id },
      { type = "Group", id = compliance_id }
    ]
  }
]
```

## Approval Entities

Each approval entity represents someone who can approve (or be notified about) an access request.

### Approval Entity Attributes

- `type` (Required, String) The type of approval entity. Valid values:
  - `"Automatic"` - Access is automatically approved without human intervention
  - `"Manager"` - The requester's direct manager must approve
  - `"ResourceOwner"` - The owner of the resource being accessed must approve
  - `"Group"` - Any member of a specific group can approve (requires `id`)
  - `"User"` - A specific user must approve (requires `id`)

- `id` (Optional, String) **Required when `type` is `"Group"` or `"User"`.** The unique identifier of the group or user.
  - Obtain user IDs from `entitle_user` data source
  - Obtain group IDs from `entitle_group` data source
  - Must be omitted for types: `"Automatic"`, `"Manager"`, `"ResourceOwner"`

### Approval Entity Behavior

**Manager Type:**
- Requires the requester to have a manager assigned in Entitle
- **If the requester has no manager assigned, the access request will fail**
- Best Practice: Ensure all users who will request access have managers assigned
- Can be combined with other entity types in the same step using `operator = "or"` to provide fallback options

**ResourceOwner Type:**
- Requires the requested resource to have an owner assigned
- **All resources/integrations in Entitle must have an owner assigned**
- The resource owner will be notified and must approve the request

**Group Type:**
- **Minimum 1 member of the group must approve**
- Any single member of the group can approve (not all members required)
- If using `operator = "and"` with multiple groups, one member from each group must approve

**User Type:**
- The specific named user must approve
- Useful for designated approvers (e.g., Security Lead, Compliance Officer)

**Automatic Type:**
- No human approval required
- Access is granted immediately upon request
- Use only for low-risk scenarios
- Still creates audit trail

### Combining Entity Types

You can combine different entity types in the same step:

```terraform
# Example: Manager OR Security Team Lead OR CISO can approve
{
  sort_order = 1
  operator = "or"
  
  approval_entities = [
    {
      type = "Manager"
    },
    {
      type = "Group"
      id   = data.entitle_group.security.id
    },
    {
      type = "User"
      id   = data.entitle_user.ciso.id
    }
  ]
}
```

This provides flexibility and ensures requests can be approved even if:
- The requester has no manager (Security or CISO can still approve)
- The manager is unavailable (Security or CISO can approve)
- Multiple approval paths exist for faster processing

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

- `id` (String) The unique identifier of the workflow (UUID format).

**Note:** Workflows do not expose additional metadata fields such as `created_at`, `updated_at`, `created_by`, `is_active`, or usage statistics. These details are available only through the Entitle UI.

## Import

Workflows can be imported using their UUID:

```shell
terraform import entitle_workflow.example a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

### Finding the Workflow ID

To find the UUID of an existing workflow:

1. Log in to the Entitle UI
2. Navigate to the **Workflows** section
3. Click on the workflow you want to import
4. The workflow ID (UUID) will be visible in the browser URL
   - Example: `https://app.entitle.io/workflows/a1b2c3d4-e5f6-7890-abcd-ef1234567890`
5. Copy the UUID from the URL

**Example URL:**
```
https://app.entitle.io/workflows/a1b2c3d4-e5f6-7890-abcd-ef1234567890
                                 └─────────────── This is the workflow ID ──────────────┘
```

### Import Limitations

There are no special limitations when importing workflows. All workflow configurations can be imported and managed via Terraform.

### After Import

After importing, run `terraform plan` to ensure the imported configuration matches your Terraform code. You may need to adjust your `.tf` files to match the existing workflow configuration.

## Notes and Best Practices

### Rule Evaluation Order

- Rules are evaluated in `sort_order` (lowest first)
- The **first matching rule** is applied
- Subsequent rules are not evaluated
- Always order rules from most specific to least specific

**Example:**
```terraform
rules = [
  {
    sort_order = 1
    under_duration = 3600      # Matches requests ≤ 1 hour
    # ... short duration approval ...
  },
  {
    sort_order = 2
    under_duration = 7200      # Matches requests ≤ 2 hours (but > 1 hour)
    # ... medium duration approval ...
  },
  {
    sort_order = 3
    under_duration = 999999999 # Catch-all for longer durations
    # ... long duration approval ...
  }
]
```

### Duration Best Practices

- Use seconds for `under_duration`:
  - 1 hour = `3600`
  - 2 hours = `7200`
  - 4 hours = `14400`
  - 8 hours = `28800`
  - 24 hours = `86400`
  - 7 days = `604800`

- Create a final catch-all rule with a very large `under_duration` (e.g., `999999999`) to handle any duration

- Order rules from shortest to longest duration for logical flow

### Schedule Considerations

- If not using time-based rules, set `any_schedule = true` and `in_schedules = []`
- Schedule-based rules are useful for:
  - Different approval requirements during business hours vs after-hours
  - Weekend access requiring additional approval
  - Maintenance windows with relaxed approval
- Multiple schedules use OR logic (matches if current time is in ANY listed schedule)

### Group-Based Rules

- Use `in_groups` to create team-specific workflows
- If `in_groups = []`, the rule applies to all users
- Users must be in at least one of the listed groups for the rule to match

### Approval Strategy: Steps vs Operators

**When to use multiple steps (sequential):**
- Approvals must happen in a specific order
- Each level of approval builds on the previous
- Example: Manager → Team Lead → Director

**When to use operator = "and" (parallel):**
- Multiple approvals needed but order doesn't matter
- Faster approval process (approvers work in parallel)
- Example: Security AND Compliance (both must approve, any order)

**When to use operator = "or":**
- Any one of several approvers is sufficient
- Provides flexibility in who can approve
- Example: Any Security team member OR Any DevOps team member

**Best Practice:**
Combine techniques for complex requirements:
```terraform
# Step 1: Manager (gates the request)
# Step 2: Security AND Compliance (parallel final approval)
steps = [
  { sort_order = 1, operator = "or",  /* Manager */ },
  { sort_order = 2, operator = "and", /* Security + Compliance */ }
]
```

### Group Approval Behavior

When using `type = "Group"`:
- **With operator = "or"**: Any ONE member of any listed group can approve
- **With operator = "and"**: Any ONE member of each listed group must approve

Example:
```terraform
{
  operator = "and"
  approval_entities = [
    { type = "Group", id = security_group },    # Any security member
    { type = "Group", id = compliance_group }   # AND any compliance member
  ]
}
```
This requires one security member AND one compliance member (not all members of both groups).

### Multi-Step Approvals

- To require multiple approvals, use multiple steps (not multiple entities with "and" operator)
- Each step must complete before the next step begins
- Use `sort_order` to define the sequence

**Example: Manager then Security Team**
```terraform
approval_flow = {
  steps = [
    { sort_order = 1, /* Manager approval */ },
    { sort_order = 2, /* Security approval */ }
  ]
}
```

### Automatic Approval

- Use sparingly and only for low-risk scenarios
- Good for:
  - Development environments
  - Short-duration access
  - Read-only access
  - Non-production resources

### Notifications

- Use `notified_entities` to keep stakeholders informed without requiring their approval
- Useful for audit trails and awareness
- Does not slow down the approval process

## Common Patterns

### Pattern 1: Progressive Approval by Duration

Longer access requests require stricter approval:

```terraform
rules = [
  {
    sort_order = 1
    under_duration = 3600  # ≤1h: Auto-approve
    approval_flow = {
      steps = [{ operator = "or", approval_entities = [{ type = "Automatic" }] }]
    }
  },
  {
    sort_order = 2
    under_duration = 14400  # 1-4h: Manager
    approval_flow = {
      steps = [{ operator = "or", approval_entities = [{ type = "Manager" }] }]
    }
  },
  {
    sort_order = 3
    under_duration = 999999999  # >4h: Manager + Security (sequential)
    approval_flow = {
      steps = [
        { sort_order = 1, operator = "or", approval_entities = [{ type = "Manager" }] },
        { sort_order = 2, operator = "or", approval_entities = [{ type = "Group", id = security_id }] }
      ]
    }
  }
]
```

### Pattern 2: Dual Approval (Parallel)

Two teams must both approve, but can happen in any order:

```terraform
approval_flow = {
  steps = [{
    operator = "and"
    approval_entities = [
      { type = "Group", id = security_id },
      { type = "Group", id = compliance_id }
    ]
  }]
}
```

### Pattern 3: Tiered Sequential Approval

Each level must approve in order:

```terraform
approval_flow = {
  steps = [
    { sort_order = 1, operator = "or", approval_entities = [{ type = "Manager" }] },
    { sort_order = 2, operator = "or", approval_entities = [{ type = "Group", id = team_lead_id }] },
    { sort_order = 3, operator = "or", approval_entities = [{ type = "Group", id = director_id }] }
  ]
}
```

### Pattern 4: Flexible Approval with Escalation

Normal approval OR automatic escalation to security:

```terraform
# Short duration: Manager OR any Security member
steps = [{
  operator = "or"
  approval_entities = [
    { type = "Manager" },
    { type = "Group", id = security_id }
  ]
}]

# Longer duration: Manager THEN Security
steps = [
  { sort_order = 1, operator = "or", approval_entities = [{ type = "Manager" }] },
  { sort_order = 2, operator = "or", approval_entities = [{ type = "Group", id = security_id }] }
]
```

### Pattern 5: Business Hours Flexibility

Relaxed approval during business hours, stricter after-hours:

```terraform
rules = [
  {
    # Business hours: Auto-approve
    in_schedules = [business_hours_id]
    approval_flow = { steps = [{ approval_entities = [{ type = "Automatic" }] }] }
  },
  {
    # After hours: Security approval required
    any_schedule = true
    approval_flow = { steps = [{ approval_entities = [{ type = "Group", id = security_id }] }] }
  }
]
```

### Pattern 6: Role-Based Workflows

Different approval chains for different teams:

```terraform
rules = [
  {
    # Developers: Auto-approve
    in_groups = [developers_id]
    approval_flow = { steps = [{ approval_entities = [{ type = "Automatic" }] }] }
  },
  {
    # Contractors: Manager approval
    in_groups = [contractors_id]
    approval_flow = { steps = [{ approval_entities = [{ type = "Manager" }] }] }
  },
  {
    # External: Manager + Security approval
    in_groups = [external_id]
    approval_flow = {
      steps = [
        { sort_order = 1, approval_entities = [{ type = "Manager" }] },
        { sort_order = 2, approval_entities = [{ type = "Group", id = security_id }] }
      ]
    }
  }
]
```

### Pattern 7: Break-Glass Access

Emergency access with automatic approval but full audit:

```terraform
approval_flow = {
  steps = [{
    operator = "or"
    approval_entities = [{ type = "Automatic" }]
    notified_entities = [
      { type = "User", id = ciso_id },
      { type = "Group", id = security_team_id },
      { type = "Group", id = compliance_team_id }
    ]
  }]
}
```

## Manager Assignment Best Practices

Since workflows using `type = "Manager"` require users to have managers assigned:

### Prevention

1. **Validate Manager Assignments**
   - Ensure all users have managers assigned before deploying Manager-based workflows
   - Regular audits of user manager assignments
   
2. **Use Conditional Workflows**
   - Create different workflows for users with and without managers:
   
   ```terraform
   data "entitle_group" "users_with_managers" {
     name = "Users With Managers"
   }
   
   data "entitle_group" "users_without_managers" {
     name = "Contractors"  # Example: contractors might not have managers
   }
   
   resource "entitle_workflow" "standard" {
     name = "Standard Manager Approval"
     
     rules = [{
       in_groups = [data.entitle_group.users_with_managers.id]
       
       approval_flow = {
         steps = [{
           approval_entities = [{ type = "Manager" }]
         }]
       }
     }]
   }
   
   resource "entitle_workflow" "no_manager" {
     name = "Team Lead Approval"
     
     rules = [{
       in_groups = [data.entitle_group.users_without_managers.id]
       
       approval_flow = {
         steps = [{
           approval_entities = [{
             type = "Group"
             id   = data.entitle_group.team_leads.id
           }]
         }]
       }
     }]
   }
   ```

3. **Provide Fallback Approvers**
   - Use `operator = "or"` to provide alternative approval paths:
   
   ```terraform
   approval_entities = [
     { type = "Manager" },
     { type = "Group", id = default_approvers_id }
   ]
   ```

