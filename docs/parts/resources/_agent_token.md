An Entitle Agent Token is a credential used to authenticate an Entitle Agent with the Entitle platform. The Entitle Agent is a lightweight service that runs inside your network and enables Entitle to connect to internal systems and applications that are not directly accessible from the internet (e.g., on-premise databases, private cloud services, or air-gapped environments).

Each agent token is associated with a single agent deployment. Once generated, the token value is only available at creation time and must be securely stored for use during agent setup. [Read more about agents](https://docs.beyondtrust.com/entitle/docs/entitle-agent).

## Key Concepts

- **Entitle Agent**: A service deployed in your environment that acts as a bridge between Entitle and your internal systems
- **Agent Token**: The authentication credential the agent uses to securely connect to the Entitle platform
- **Token Value**: The sensitive secret returned only at creation — store it securely immediately after applying
- **One Token Per Agent**: Each token authenticates exactly one agent deployment (tokens cannot be shared between agents unless they are fully redundant)

## When to Use Agent Tokens

- Connecting Entitle to internal/private integrations that are behind a firewall or VPN
- Setting up redundant agent deployments for high availability
- Rotating agent credentials as part of a security policy
- Managing multiple agent deployments via Terraform (one token per agent)

## Example Usage

### Basic Agent Token

Create a token for a single agent deployment:

```terraform
resource "entitle_agent_token" "primary_agent" {
  name = "primary-agent"
}

# Save the token value securely after apply
output "agent_token_value" {
  value     = entitle_agent_token.primary_agent.token
  sensitive = true
}
```

### Agent Token for a Specific Environment

Name tokens clearly to indicate their purpose or environment:

```terraform
resource "entitle_agent_token" "production_agent" {
  name = "production-datacenter-agent"
}

resource "entitle_agent_token" "staging_agent" {
  name = "staging-environment-agent"
}
```

### Agent Token Passed to a Kubernetes Deployment

Provision a token and pass it directly to the agent's Kubernetes configuration:

```terraform
resource "entitle_agent_token" "k8s_agent" {
  name = "kubernetes-cluster-agent"
}

resource "kubernetes_secret" "entitle_agent_secret" {
  metadata {
    name      = "entitle-agent-token"
    namespace = "entitle"
  }

  data = {
    ENTITLE_TOKEN = entitle_agent_token.k8s_agent.token
  }
}
```

### Agent Token Stored in AWS Secrets Manager

Securely store the generated token in AWS Secrets Manager:

```terraform
resource "entitle_agent_token" "aws_agent" {
  name = "aws-private-vpc-agent"
}

resource "aws_secretsmanager_secret" "entitle_token" {
  name        = "entitle/agent-token"
  description = "Entitle Agent authentication token"
}

resource "aws_secretsmanager_secret_version" "entitle_token_value" {
  secret_id     = aws_secretsmanager_secret.entitle_token.id
  secret_string = entitle_agent_token.aws_agent.token
}
```

### Integration Using an Agent Token

Link an agent token to an integration that requires agent-based connectivity:

```terraform
resource "entitle_agent_token" "db_agent" {
  name = "internal-database-agent"
}

resource "entitle_integration" "internal_postgres" {
  name            = "Internal PostgreSQL"
  connection_json = jsonencode({
    host     = "postgres.internal.example.com"
    port     = 5432
    database = "production"
  })

  application = {
    name = "postgresql"
  }

  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  agent_token = {
    name = entitle_agent_token.db_agent.name
  }

  allowed_durations       = [3600, 28800]
  allow_creating_accounts = false
}
```

## Attributes Reference

### Required

- `name` (String) A descriptive display name for the agent token. Use a name that identifies the agent's purpose or environment (e.g., `"production-datacenter-agent"`, `"kubernetes-cluster-agent"`).

### Read-Only

- `id` (String) The unique identifier of the agent token (UUID format).
- `token` (String, **Sensitive**) The authentication token value. **This value is only available at creation time.** After the initial `terraform apply`, it cannot be retrieved again. Store it securely immediately — for example, in a secrets manager like AWS Secrets Manager, HashiCorp Vault, or Azure Key Vault.

## Import

Existing agent tokens can be imported using their UUID:

```shell
terraform import entitle_agent_token.example a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

**Note:** Importing an agent token does not recover the `token` value. The `token` attribute will be empty after import. Only use import to bring an existing token resource under Terraform management — you cannot retrieve the token secret via import.

### Finding the Agent Token ID

To find the UUID of an existing agent token:

1. Log in to the Entitle UI
2. Navigate to **Org Settings** → **Agent Tokens**
3. Locate the token you want to import
4. The token ID (UUID) will be visible in the UI or the browser URL

## Notes and Best Practices

### Token Security

- The `token` output is marked `sensitive = true` — it will not appear in plan output by default
- **Store the token immediately after `terraform apply`** — it cannot be retrieved after the initial creation
- Never commit raw token values to source control
- Use a secrets manager (AWS Secrets Manager, HashiCorp Vault, Azure Key Vault) to store and distribute token values to agent deployments

### One Token Per Agent

- A single agent token should be used by only one agent deployment unless the agents are fully redundant (active-passive failover)
- For multiple independent agent deployments, create separate tokens with distinct names

### Token Rotation

- To rotate a token, create a new `entitle_agent_token` resource with a new name, update the agent deployment to use the new token, then delete the old resource
- Entitle generates new token values on each resource creation — there is no in-place rotation

### Agent Deployment

- After creating the token in Terraform, use the `token` output to configure the agent
- The agent is typically deployed as a Docker container or Kubernetes pod in your private network
- Refer to the [Terraform-based agent installation guide](https://docs.beyondtrust.com/entitle/docs/terraform-based-agent-installation-guide) for full setup instructions
