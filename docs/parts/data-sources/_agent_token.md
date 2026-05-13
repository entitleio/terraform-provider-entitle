An Entitle Agent Token is the authentication credential used by an Entitle Agent to connect to the Entitle platform. The Entitle Agent is a service deployed inside your private network that enables Entitle to reach internal systems not accessible from the internet (e.g., on-premise databases, VPC-internal services, or air-gapped environments).

Use this data source to look up an existing agent token by ID — for example, to retrieve its name for reference in an `entitle_integration` resource, or to inspect its configuration as part of an audit. [Read more about agents](https://docs.beyondtrust.com/entitle/docs/entitle-agent).

## Key Concepts

- **Agent Token**: The credential that authenticates an agent deployment with Entitle
- **Token Value**: The sensitive secret — only available at creation time via the `entitle_agent_token` resource; **this data source does not return the token value**
- **Name**: The human-readable label for the token, used when associating a token with an integration

## When to Use This Data Source

- Looking up an agent token's name when you know its ID (e.g., to reference in an integration configuration)
- Verifying that a specific agent token exists as part of a configuration audit
- Retrieving token metadata without needing the secret value

## Example Usage

### Look Up an Agent Token by ID

```terraform
data "entitle_agent_token" "my_agent" {
  id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

output "agent_token_name" {
  value = data.entitle_agent_token.my_agent.name
}
```

### Reference an Existing Agent Token in an Integration

If an agent token was created outside of your current Terraform configuration, look it up and reference it in an integration:

```terraform
data "entitle_agent_token" "existing_agent" {
  id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

resource "entitle_integration" "internal_service" {
  name            = "Internal Service"
  connection_json = jsonencode({
    endpoint = "https://service.internal.example.com"
    api_key  = var.service_api_key
  })

  application = {
    name = "custom"
  }

  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120003"
  }

  agent_token = {
    name = data.entitle_agent_token.existing_agent.name
  }

  allowed_durations       = [3600, 28800]
  allow_creating_accounts = false
}
```

## Query Parameters

### Required

- `id` (String) The unique identifier of the agent token to retrieve (UUID format).

## Returned Attributes

- `id` (String) The agent token's unique identifier (UUID format).
- `name` (String) The display name of the agent token.

**Note:** The sensitive `token` value is **not** returned by this data source. The token secret is only available immediately after creation via the `entitle_agent_token` resource. Store it securely at creation time.

## Finding Agent Token IDs

To find the UUID of an existing agent token:

1. Log in to the Entitle UI
2. Navigate to **Org Settings** → **Agent Tokens**
3. Locate the token you need
4. The token ID (UUID) is visible in the UI or the browser URL

## Notes

- This data source does not expose the secret `token` value — use it only for metadata lookups (name, ID)
- If you need to create a new token and obtain the secret, use the `entitle_agent_token` resource instead
- The token `name` returned here can be used to configure `agent_token.name` in `entitle_integration` resources
