package docs

import _ "embed"

var (
	//go:embed parts/_index.md
	ProviderMarkdownDescription string
)

// data sources
var (
	//go:embed parts/data-sources/_access_request_forward.md
	AccessRequestForwardDataSourceMarkdownDescription string
	//go:embed parts/data-sources/_access_review_forward.md
	AccessReviewForwardDataSourceMarkdownDescription string
	//go:embed parts/data-sources/_accounts.md
	AccountsDataSourceMarkdownDescription string
	//go:embed parts/data-sources/_agent_token.md
	AgentTokenDataSourceMarkdownDescription string
	//go:embed parts/data-sources/_bundle.md
	BundleDataSourceMarkdownDescription string
	//go:embed parts/data-sources/_directory_groups.md
	DirectoryGroupsDataSourceMarkdownDescription string
	//go:embed parts/data-sources/_integration.md
	IntegrationDataSourceMarkdownDescription string
	//go:embed parts/data-sources/_permissions.md
	PermissionsDataSourceMarkdownDescription string
	//go:embed parts/data-sources/_policy.md
	PolicyDataSourceMarkdownDescription string
	//go:embed parts/data-sources/_resource.md
	ResourceDataSourceMarkdownDescription string
	//go:embed parts/data-sources/_resources.md
	ResourcesDataSourceMarkdownDescription string
	//go:embed parts/data-sources/_role.md
	RoleDataSourceMarkdownDescription string
	//go:embed parts/data-sources/_roles.md
	RolesDataSourceMarkdownDescription string
	//go:embed parts/data-sources/_user.md
	UserDataSourceMarkdownDescription string
	//go:embed parts/data-sources/_users.md
	UsersDataSourceMarkdownDescription string
	//go:embed parts/data-sources/_workflow.md
	WorkflowDataSourceMarkdownDescription string
)

// resources
var (
	//go:embed parts/resources/_access_request_forward.md
	AccessRequestForwardResourceMarkdownDescription string
	//go:embed parts/resources/_access_review_forward.md
	AccessReviewForwardResourceMarkdownDescription string
	//go:embed parts/resources/_agent_token.md
	AgentTokenResourceMarkdownDescription string
	//go:embed parts/resources/_bundle.md
	BundleResourceMarkdownDescription string
	//go:embed parts/resources/_integration.md
	IntegrationResourceMarkdownDescription string
	//go:embed parts/resources/_permission.md
	PermissionResourceMarkdownDescription string
	//go:embed parts/resources/_policy.md
	PolicyResourceMarkdownDescription string
	//go:embed parts/resources/_resource.md
	ResourceResourceMarkdownDescription string
	//go:embed parts/resources/_role.md
	RoleResourceMarkdownDescription string
	//go:embed parts/resources/_workflow.md
	WorkflowResourceMarkdownDescription string
)
