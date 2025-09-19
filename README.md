# Argo CD MCP

Argo CD MCP is a Model Context Protocol Server to converse with Argo CD from a UI such as Anthropic's Claude or Block's Goose

## Features

- Prompts:
  - `argocd-unhealthy-application-resources`: list the Unhealthy (`Degraded` and `Progressing`) Applications in Argo CD
- Tools:
  - `unhealthyApplications`: list the Unhealthy (`Degraded` and `Progressing`) Applications in Argo CD
  - `unhealthyApplicationResources`: list unhealthy resources of a given Argo CD Application

Example:

> list the unhealthy applications on Argo CD and for each one, list their unhealthy resources


## Building and Installing

Requires [Go 1.24 (or higher)](https://go.dev/doc/install) and [Task](https://taskfile.dev/)

Build the binary with the following command:
```
task install
```

Build the Container image with the following command:
```
task build-image
```


## Using the Argo CD MCP Server

### Obtaining a token to connect to Argo CD

Create a local account in Argo CD with `apiKey` capabilities only (not need for `login`). See [Argo CD documentation for more information](https://argo-cd.readthedocs.io/en/stable/operator-manual/user-management/). 
Once create, generate a token via the 'Settings > Accounts' page in the Argo CD UI or via the `argocd account generate-token` command and store the token in a `token-file` which will be passed as an argument when running the server (see below).

### Stdio Transport with Claude Desktop App

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
                "stdio",
                "--argocd-token"
                "<token>",
                "--argocd-url",
                "<url>",
                "--insecure",
                "<true|false>",
                "debug",
                "<true|false>"
            ]
        }
    }
}
```

### Stdio Transport in Cursor

Edit your `~/.cursor/mcp.json` file with the following contents:

```
{
  "mcpServers": {
    "argocd-mcp": {
      "command": "<path/to/argocd-mcp>",
      "args": [
        "stdio",
        "--argocd-token",
        "<token>",
        "--argocd-url",
        "<url>",
        "--insecure",
        "<true|false>",
        "--debug",
        "<true|false>"
      ]
    }
  }
}
```

### HTTP Transport with Cursor

Start the Argo CD MCP server from the binary after running `task install`:

```
argocd-mcp http --argocd-url=<url> --argocd-token=<token> --debug=<true|false> --port=<port>
```

Or start the Argo CD MCP server as a container after running `task build-image`:

```
podman run -d --name argocd-mcp -e ARGOCD_MCP_URL=<url> -e ARGOCD_MCP_TOKEN=<token> -e ARGOCD_MCP_DEBUG=<true|false> -p 8080:8080 argocd-mcp:latest
```

Edit your `~/.cursor/mcp.json` file with the following contents:

```
{
  "mcpServers": {
    "argocd-mcp": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

## License

The code is available under the Apache License 2.0
