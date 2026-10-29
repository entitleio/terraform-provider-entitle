The Entitle provider allows you to manage your [Entitle](https://www.entitle.io) access, workflows, integrations, and policies within your Entitle environment using Infrastructure as Code.

## Use Cases
- **Just-in-Time Access**: Grant temporary elevated permissions for specific tasks
- **Break-Glass Access**: Provide emergency access with proper approval and audit trails
- **Onboarding/Offboarding**: Automate permission grants and revocations based on group membership
- **Compliance**: Maintain audit trails and enforce approval workflows
- **Cross-Application Access**: Bundle permissions across AWS, GCP, Azure, and SaaS applications

## Authentication

The provider requires an API key to authenticate with the Entitle API.

### Getting Your API Key

To obtain an API key:

1. Log in to your Entitle account
2. Navigate to **Org Settings** → **Tokens**
3. Click **Add**
4. Provide a name for the token
5. Copy the token immediately (it will only be shown once)
6. Store it securely in your secret management system

→ [View detailed instructions](https://docs.beyondtrust.com/entitle/docs/org-settings#tokens)

**Important**: API tokens are organization-level credentials. Treat them as sensitive secrets and never commit them to version control.

### Configuring the API Key

The API key can be provided in two ways:

**Option 1: Provider configuration block**
```terraform
provider "entitle" {
  api_key = var.entitle_api_key
}
```

**Option 2: Environment variable**
```bash
export ENTITLE_API_KEY="your-api-key-here"
```

### Regional Endpoints

Entitle provides different API endpoints based on your organization's region:

| Region           | Endpoint URL                | Usage                     |
|------------------|-----------------------------|---------------------------|
| **EU (Default)** | `https://api.entitle.io`    | European region (default) |
| **US**           | `https://api.us.entitle.io` | United States region      |
| **CA**           | `https://api.ca.entitle.io` | Canada region             |

**How to determine your region:**
- Check your Entitle login URL or contact your Entitle administrator
- Your organization's region is determined during initial setup
- If unsure, the default EU endpoint is `https://api.entitle.io`

The following environment variables can be used as an alternative to provider configuration:

| Environment Variable | Description |
|---------------------|-------------|
| `ENTITLE_API_KEY` | API key for authentication |
| `ENTITLE_API_ENDPOINT` | API endpoint URL |
| `ENTITLE_BULK_CHUNK_SIZE` | Maximum number of items per bulk API request (default `1000`) |
| `ENTITLE_BULK_PARALLELISM` | Maximum number of concurrent bulk API requests (default `4`) |

### Bulk Operations

Bulk resources such as `entitle_resource_tags` and `entitle_resource_settings` change many entities with asynchronous bulk API jobs. The optional `bulk` block sets limits for all of them:

```terraform
provider "entitle" {
  bulk {
    chunk_size  = 1000 # items per bulk request
    parallelism = 4    # concurrent requests
  }
}
```

- `chunk_size` splits large ID lists into several bulk requests.
- `parallelism` limits concurrent API requests, including the batched reads (100 IDs per request) bulk resources perform on refresh.
- Entitle runs one bulk operation per tenant at a time. The provider waits and retries while another one is running, so bulk jobs themselves run one after another.
- Values set in the block take precedence over the environment variables; both default to the values shown above.
