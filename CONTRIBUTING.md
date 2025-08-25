# Contributing to the Entitle Terraform Provider

Thank you for your interest in contributing to our project!

Here is some information on how to get started and where to ask for help.

## Getting Started

The Entitle Terraform provider is a translation layer between Terraform and the Entitle API. Thus, the documentation for the Entitle API directly applies to the Terraform resources defined. All Terraform schemas should directly map to an Entitle API endpoint. Not all endpoints in the Entitle API have Terraform counterparts.

The Entitle API documentation is available from your Entitle environment under **Settings > API Access**. You must be logged in with administrator permissions to view and configure API access.

## How can I Contribute?

### Reporting Bugs

Bugs should be submitted through Entitle Support. Please open a ticket in the support portal with details about the issue, including any error logs, Terraform configurations, or Entitle environment details that may help reproduce the problem. Our support team will ensure the escalation is raised to the proper engineering team.

If the bug is a security vulnerability, please instead refer to the [responsible disclosure section of our security policy](https://entitle.io/security).

### Feature Requests

Feature requests should also be submitted through Entitle Support. Submitting through our support organization will ensure the request gets routed to the proper Product Management team for consideration.

### Making Changes and Submitting a Pull Request

#### **Did you write a patch that fixes a bug?**

- Open a GitHub pull request with the patch.
- Ensure the PR description clearly describes both the problem and the solution. If you have a support ticket, please include that number as well.
- We will review the changes and make a determination if we will accept the change.

#### **Do you intend to add a new feature or change an existing one?**

- Consider submitting a feature request through Entitle Support to ensure your proposed changes do not conflict with planned or in-development features.
- If you do open a PR, please ensure the description clearly describes what the change is, and what problem it solves.
- Any new code must include unit tests (if possible) or end-to-end tests (if Terraform resources are changed or added). All tests must pass before the change can be merged.
- We will review the change and determine if it fits within our goals for the project.

### Tests

All tests must pass for any submitted changes to be accepted. This includes the acceptance tests defined in the repository.

#### Running Acceptance Tests

The Entitle Terraform Provider includes end-to-end acceptance tests that exercise the actual behavior of the provider against real Entitle infrastructure. These tests are implemented in Go (using the Terraform plugin-testing framework) and are intended to verify create/read/update/delete operations against actual Entitle API endpoints.

To run them:

```sh
make testacc
```

Tests will create and destroy real resources in your Entitle environment. Do not run these tests against a production environment.
