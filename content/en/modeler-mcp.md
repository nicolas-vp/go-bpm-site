# AI agent in the modeler

[go-bpm modeler](/en/modeler/) includes an MCP server. Claude Code, Cursor and other MCP clients can inspect open diagrams, including unsaved edits and selected elements, explain errors and propose changes. You approve each change in the modeler.

## Requirements

Use a modeler build with MCP support, available from the dev channel since 31 October 2026, an HTTP-capable MCP client, and an open workspace such as your service's `process/` directory. The client and modeler run on the same computer.

## Enable the server

1. Launch the modeler and select a workspace folder.
2. Click the AI agent button on the toolbar.
3. Enable the MCP server. The dialog shows its address, connection command and token; the toolbar button shows a green indicator.

The enabled state is remembered for the next launch. Disable it with the same checkbox. The token authenticates the agent; generating a new token immediately invalidates the previous one.

## Connect a client

### Claude Code

Copy the command from the dialog:

```sh
claude mcp add --transport http go-bpm-modeler http://127.0.0.1:39190/mcp \
  --header "Authorization: Bearer <token from the dialog>"
```

Without `--scope`, the server belongs to the current project. Add `--scope user` for all projects. Check with `claude mcp list` or `/mcp` in Claude Code.

### Cursor

Add to the project's `.cursor/mcp.json` or your user-level `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "go-bpm-modeler": {
      "url": "http://127.0.0.1:39190/mcp",
      "headers": {"Authorization": "Bearer <token from the dialog>"}
    }
  }
}
```

### Other clients

Use Streamable HTTP at `http://127.0.0.1:39190/mcp` with the bearer authorization header. Responses use ordinary JSON, without an SSE stream.

## Work together

On connection, the agent is instructed to read the engine's modeling guide before inspecting the editor. Ask about the open tab or selected elements:

- “Explain the linter errors in this process.”
- “Which data model fields does the selected task need?”
- “Which Offer rule matches a gold customer ordering 12,000?”
- “Add an escalation if approval takes more than three days.”
- “Replace random activity IDs with meaningful names.”
- “Create an order refund process in orders/refund.bpmn.”

Selection acts as a pointer: the agent can inspect exactly the selected elements.

## Approving edits

For a diagram update or new file, the modeler asks which file will change, what changes and why. Accepting an update loads it into the editor and marks the tab unsaved. You choose whether to save it with Ctrl+S. Rejecting leaves the current diagram unchanged.

Invalid XML is not applied; the agent receives an error and can try again. After an edit, review the linter results. An update replaces the entire diagram and cannot be undone with Ctrl+Z; close without saving if you want to discard it.

## Agent tools

| Tool | Purpose | Approval |
|---|---|---|
| `get_modeling_guide` | Supported elements and properties | No |
| `list_files` | Workspace BPMN and DMN files | No |
| `read_file` | Saved workspace file | No |
| `list_open_diagrams` | Tabs, active file and unsaved state | No |
| `get_diagram` | Current XML, including unsaved changes | No |
| `describe_diagram` | Elements, properties, flows, DMN and lint issues | No |
| `get_selection` | Selected elements | No |
| `update_diagram` | Replace a diagram in a tab | Yes |
| `create_diagram` | Create and open a workspace diagram | Yes |

Tab tools accept a `path`; without it they use the active tab. An unopened file can be opened by the modeler before an update.

## Access boundaries

The server is off until enabled, listens on `127.0.0.1`, requires a token and rejects browser-page requests through Host/Origin checks. File access is limited to the selected workspace. The token is stored in user-specific modeler settings. Editing requires approval, and saving an updated diagram remains your action.

Closing the modeler removes live editor access. A client may still have its own unrelated filesystem tools.

## Troubleshooting

| Symptom | Action |
|---|---|
| Port 39190 is occupied | Set `"mcpPort":39191` in modeler settings, restart, then use the new dialog address |
| HTTP 401 | Reconnect with the current token |
| Connection refused | Start the modeler and enable its server |
| No workspace | Select a folder |
| Diagram not open | Open it in a tab or ask for the saved file |
| Editor did not respond | Close other modal dialogs and retry |

Settings paths:

| OS | Path |
|---|---|
| Linux | `~/.config/go-bpm-modeler/settings.json` |
| macOS | `~/Library/Application Support/go-bpm-modeler/settings.json` |
| Windows | `%AppData%\go-bpm-modeler\settings.json` |
