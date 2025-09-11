# Argo CD MCP

Argo CD MCP is a Model Context Protocol Server to converse with Argo CD from a UI such as Anthropic's Claude or Block's Goose

## Available Prompts

1. `argocd-unhealthy-application-resources`: list the Unhealthy (`Degraded` and `Progressing`) Applications in Argo CD

## Available Tools

1. `unhealthyApplications`: list the Unhealthy (`Degraded` and `Progressing`) Applications in Argo CD
2. `unhealthyApplicationResources`: list unhealthy resources of a given Argo CD Application

Example:

> list the unhealthy applications on Argo CD and for each one, list their unhealthy resources


## Building and Installing

Requires [Go 1.24 (or higher)](https://go.dev/doc/install) and [Task](https://taskfile.dev/)

```
task install
```

## Testing

### Obtaining a token to connect to Argo CD

Create a local account in Argo CD with `apiKey` capabilities only (not need for `login`). See [Argo CD documentation for more information](https://argo-cd.readthedocs.io/en/stable/operator-manual/user-management/). 
Once create, generate a token via the 'Settings > Accounts' page in the Argo CD UI or via the `argocd account generate-token` command and store the token in a `token-file` which will be passed as an argument when running the server (see below).

### Testing with Cursor

Edit your `~/.cursor/mcp.json` file with the following contents:

```
{
  "mcpServers": {
    "argocd-mcp": {
      "command": "<path/to/argocd-mcp>",
      "args": [
        "--argocd-token-file",
        "<path/to/token-file>",
        "--argocd-url",
        "<url>",
        "--insecure",
        "<true|false>"
      ]
    }
  }
}
```

## Testing with Goose CLI or UI

[Install Goose](https://block.github.io/goose/docs/getting-started/installation) then [add the MCP server](https://block.github.io/goose/docs/getting-started/using-extensions#mcp-servers) with the following command line to run:

```
<path/to/argocd-mcp> --argocd-token-file <path/to/token-file> --argocd-url <url> --insecure <true|false>
```

## Testing with Claude Desktop App

On macOS, run the following command:

```
code ~/Library/Application\ Support/Claude/claude_desktop_config.json
```

and add the following MCP server definition:
```
{
    "mcpServers": {
        "argocd-mcp": {
            "command": "<path/to/argocd-mcp>",
            "args": [
                "--argocd-token-file"
                "<path/to/token-file>",
                "--argocd-url",
                "<url>",
                "--insecure"
                "<true|false>"
            ]
        }
    }
}
```

## License

The code is available under the Apache License 2.0
