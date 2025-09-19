## MCP server

> :warning: Warning! This project is in its very early stages of development. Expect frequent changes and potential breaking updates as we iterate on features and architecture.

The MCP server allows the [Rancher AI agent](https://github.com/rancher-sandbox/rancher-ai-agent) to securely retrieve or update Kubernetes and Rancher resources across local and downstream clusters. It expects the Rancher token in a header, which the agent will always provide for authentication.
