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
2. Navigate to **Organization Settings** → **Tokens**
3. Click **Create Token** or **Generate New Token**
4. Provide a name/description for the token
5. Copy the token immediately (it will only be shown once)
6. Store it securely in your secret management system

→ [View detailed instructions](https://docs.beyondtrust.com/entitle/docs/org-settings#view-and-manage-tokens)

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

| Region | Endpoint URL | Usage |
|--------|--------------|-------|
| **EU (Default)** | `https://api.entitle.io` | European region (default) |
| **US** | `https://api.us.entitle.io` | United States region |

**How to determine your region:**
- Check your Entitle login URL or contact your Entitle administrator
- Your organization's region is determined during initial setup
- If unsure, the default EU endpoint is `https://api.entitle.io`

The following environment variables can be used as an alternative to provider configuration:

| Environment Variable | Description |
|---------------------|-------------|
| `ENTITLE_API_KEY` | API key for authentication |
| `ENTITLE_ENDPOINT` | API endpoint URL |