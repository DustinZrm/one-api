# One MCP - 统一 MCP 网关

[English](README.md) | [简体中文](README_zh.md)

One MCP 是一个强大的 Model Context Protocol (MCP) 管理与分发系统。它作为一个统一网关，将多个上游 MCP 服务（支持 SSE、Stdio 和 HTTP/REST）聚合到一个标准的 MCP 端点中。

通过 One MCP，您可以集中管理 AI 工具，通过 API 密钥提供细粒度的访问控制，并为下游客户端（如 Claude Desktop、Cursor 等）提供标准化的接口。

## ✨ 功能特性

- **统一聚合**: 将来自多个源的工具组合到一个 SSE 端点中。
- **多协议支持**:
  - **SSE**: 连接到标准的 SSE MCP 服务。
  - **Stdio**: 将本地命令/脚本作为 MCP 服务运行。
  - **HTTP/REST**: 零代码将任何 REST API 封装为 MCP 工具。
- **细粒度访问控制**:
  - **服务级**: 限制密钥仅能访问特定的上游服务。
  - **工具级**: 细化到单个工具的权限控制。
- **可视化仪表盘**: 基于 React 的现代化 UI，用于管理服务、浏览工具和处理密钥。
- **安全认证**: 仪表盘内置 JWT 认证，MCP 客户端使用 Bearer Token 认证。
- **单文件部署**: Go 后端内置 React 前端，简化部署流程。

## 🚀 快速开始

### 环境要求

- **Go**: 1.23 或更高版本
- **Node.js**: 18 或更高版本（仅用于构建前端）

### 安装步骤

1. **克隆仓库**
   ```bash
   git clone https://github.com/DustinZrm/one-api.git
   cd one-api
   ```

2. **构建前端**
   ```bash
   cd web
   npm install
   npm run build
   cd ..
   ```

3. **构建并运行后端**
   ```bash
   cd server
   go mod tidy
   go build -o one-mcp cmd/server/main.go
   ./one-mcp
   ```

   服务将在 `http://localhost:8080` 启动。

## 🐳 Docker

- 从 GHCR 拉取镜像
  - `docker pull ghcr.io/DustinZrm/one-api:latest`
- 本地运行
  - `docker run -d -p 8080:8080 --name one-mcp ghcr.io/DustinZrm/one-api:latest`
- 启用数据持久化
  - `docker run -d -p 8080:8080 -v one-mcp-data:/app/server --name one-mcp ghcr.io/DustinZrm/one-api:latest`
  - SQLite 数据库 `one-mcp.db` 位于 `/app/server`（挂载卷 `one-mcp-data`）
- 环境变量
  - `GIN_MODE=release`（默认开启）
  - 如果上游服务需要代理，可加入 `HTTP_PROXY`/`HTTPS_PROXY`
- 可选：Docker Compose
  - ```yaml
    services:
      one-mcp:
        image: ghcr.io/DustinZrm/one-api:latest
        container_name: one-mcp
        ports:
          - "8080:8080"
        volumes:
          - one-mcp-data:/app/server
        environment:
          - GIN_MODE=release
    volumes:
      one-mcp-data:
    ```
  - 使用 `docker compose up -d` 启动

## 📖 使用指南

### 1. 访问仪表盘
在浏览器中打开 `http://localhost:8080`。
- **默认账号**: `admin` / `admin`
- *请在登录后立即修改密码。*

### 2. 添加上游服务
进入 **服务管理** 页面添加工具源：

- **SSE 模式**: 连接现有的 MCP 服务（如 Smithery）。
  - URL: `http://localhost:3000/sse`
- **Stdio 模式**: 运行本地 MCP 服务（如 `@modelcontextprotocol/server-filesystem`）。
  - 命令: `npx`
  - 参数: `["-y", "@modelcontextprotocol/server-filesystem", "/path/to/files"]`
- **HTTP 模式**: 将 REST API 封装为工具。
  - URL: `https://api.weather.com/v1/current`
  - 方法: `GET`
  - 参数: 可视化定义查询参数。

### 3. 创建 API 密钥
进入 **密钥管理** 页面：
- 为客户端创建密钥（例如 "Cursor Team A"）。
- 选择 **权限范围**:
  - **按服务**: 允许访问所选服务中的所有工具。
  - **按工具**: 选择允许该密钥访问的具体工具。

### 4. 连接客户端
配置您的 MCP 客户端（Claude Desktop, Cursor 等）使用 One MCP：

- **类型**: SSE
- **URL**: `http://localhost:8080/mcp/sse`
- **Headers**: `Authorization: Bearer sk-your-generated-key`

## 🛠 技术栈

- **后端**: Go (Gin, GORM, SQLite)
- **前端**: React, TypeScript, Ant Design, Vite
- **协议**: Model Context Protocol (JSON-RPC 2.0 over SSE)

## 📄 许可证

MIT License
