# One MCP - Unified MCP Gateway

[English](README.md) | [简体中文](README_zh.md)

One MCP is a powerful Model Context Protocol (MCP) management and distribution system. It acts as a unified gateway that aggregates multiple upstream MCP servers (supporting SSE, Stdio, and HTTP/REST) into a single standard MCP endpoint.

With One MCP, you can centrally manage your AI tools, provide granular access control via API keys, and offer a standardized interface for downstream clients (like Claude Desktop, Cursor, etc.).

## ✨ Features

- **Unified Aggregation**: Combines tools from multiple sources into a single SSE endpoint.
- **Multi-Protocol Support**:
  - **SSE**: Connect to standard SSE MCP servers.
  - **Stdio**: Execute local commands/scripts as MCP servers.
  - **HTTP/REST**: Wrap any REST API into an MCP tool with zero code.
- **Granular Access Control**:
  - **Server-Level**: Restrict keys to specific upstream servers.
  - **Tool-Level**: Fine-grained permissions down to individual tools.
- **Visual Dashboard**: A polished React-based UI for managing servers, viewing tools, and handling keys.
- **Secure Authentication**: Built-in JWT authentication for the dashboard and Bearer Token auth for MCP clients.
- **Single Binary Deployment**: The Go backend serves the React frontend, simplifying deployment.

## 🚀 Getting Started

### Prerequisites

- **Go**: 1.23 or higher
- **Node.js**: 18 or higher (for building frontend)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/DustinZrm/one-api.git
   cd one-api
   ```

2. **Build the Frontend**
   ```bash
   cd web
   npm install
   npm run build
   cd ..
   ```

3. **Build and Run the Backend**
   ```bash
   cd server
   go mod tidy
   go build -o one-mcp cmd/server/main.go
   ./one-mcp
   ```

   The server will start at `http://localhost:8080`.

## 📖 Usage Guide

### 1. Access the Dashboard
Open `http://localhost:8080` in your browser.
- **Default Login**: `admin` / `admin`
- *Please change your password immediately after logging in.*

### 2. Add Upstream Servers
Go to the **Servers** page to add your tool sources:

- **SSE Mode**: Connect to existing MCP servers (e.g., Smithery).
  - URL: `http://localhost:3000/sse`
- **Stdio Mode**: Run local MCP servers (e.g., `@modelcontextprotocol/server-filesystem`).
  - Command: `npx`
  - Args: `["-y", "@modelcontextprotocol/server-filesystem", "/path/to/files"]`
- **HTTP Mode**: Wrap a REST API as a tool.
  - URL: `https://api.weather.com/v1/current`
  - Method: `GET`
  - Parameters: Define query params visually.

### 3. Create API Keys
Go to the **API Keys** page:
- Create a key for your client (e.g., "Cursor Team A").
- Select **Permission Scope**:
  - **By Server**: Allow access to all tools in selected servers.
  - **By Tool**: Select specific tools allowed for this key.

### 4. Connect Clients
Configure your MCP client (Claude Desktop, Cursor, etc.) to use One MCP:

- **Type**: SSE
- **URL**: `http://localhost:8080/mcp/sse`
- **Headers**: `Authorization: Bearer sk-your-generated-key`

## 🛠 Tech Stack

- **Backend**: Go (Gin, GORM, SQLite)
- **Frontend**: React, TypeScript, Ant Design, Vite
- **Protocol**: Model Context Protocol (JSON-RPC 2.0 over SSE)

## 📄 License

MIT License
