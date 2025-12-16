# One MCP - Unified MCP Gateway

One MCP is a management and distribution system for Model Context Protocol (MCP) tools. It aggregates multiple upstream MCP servers into a single endpoint, providing unified access control and key management.

## Features

- **Unified Interface**: Aggregates tools from multiple MCP servers into a single SSE endpoint.
- **Key Management**: Generate API keys for clients.
- **Access Control**: Control which upstream servers each API key can access.
- **Web Dashboard**: Graphic interface to manage servers and keys.

## Prerequisites

- Go 1.23+
- Node.js 18+
- SQLite (Embedded)

## Getting Started

### Backend

1. Navigate to `server` directory:
   ```bash
   cd server
   ```
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Run the server:
   ```bash
   go run cmd/server/main.go
   ```
   The server will start at `http://localhost:8080`.

### Frontend

1. Navigate to `web` directory:
   ```bash
   cd web
   ```
2. Install dependencies:
   ```bash
   npm install
   ```
3. Run the development server:
   ```bash
   npm run dev
   ```
   Access the dashboard at `http://localhost:5173`.

## Usage

1. Open the dashboard.
2. Go to **Servers** page and add your upstream MCP servers (SSE URL required).
   - Example Name: `github` (Tools will be prefixed as `github__toolname`)
   - Example URL: `http://localhost:3000/sse`
3. Go to **API Keys** page and create a key.
   - Select which servers this key can access.
4. Use the key in your MCP Client (e.g. Claude Desktop, Cursor):
   - Server URL: `http://localhost:8080/mcp/sse`
   - Header: `Authorization: Bearer <YOUR_KEY>`
