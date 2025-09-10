# Argo CD MCP

Argo CD MCP is a Model Context Protocol Server to converse with Argo CD from a UI such as Anthropic's Claude or Block's Goose

## Available Prompts

1. `argocd-unhealthy-application-resources`: list the Unhealthy (`Degraded` and `Progressing`) Applications in Argo CD

## Available Tools

1. `unhealthyApplications`: list the Unhealthy (`Degraded` and `Progressing`) Applications in Argo CD
2. `unhealthyApplicationResources`: list unhealthy resources of a given Argo CD Application

Example:

> could you list the unhealthy resources in the 'example' application on Argo CD?


## Building and Installing

Requires Go 1.24 and [Task](https://taskfile.dev/)

```
task build
```

## Testing the server with Goose CLI or UI

[Install Goose](https://block.github.io/goose/docs/getting-started/installation) then [add the MCP server](https://block.github.io/goose/docs/getting-started/using-extensions#mcp-servers) with the following command line to run:

`argocd-mcp --argocd-token-file <path/to/token-file> --argocd-url <url>`

## Testing the server with Claude AI Desktop App

On macOS, run the following command:

```
code ~/Library/Application\ Support/Claude/claude_desktop_config.json
```

and add the following MCP server definition:
```
{
    "mcpServers": {
        "argocd-mcp": {
            "command": "argocd-mcp",
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
