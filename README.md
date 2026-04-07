# MCP 网关

## 描述

MCP 网关是一个反向代理服务器，它将客户端的请求转发到 MCP 服务器，或者通过统一门户使用网关下的所有 MCP 服务器。

## 特性

- 部署多个 MCP 服务器
- 连接到 MCP 服务器
- 使用网关调用 MCP 服务器
- 获取所有 MCP 服务器的 SSE 流
- 获取所有 MCP 服务器的工具

## 安装

1. 拉取 GitHub 包

```bash
docker pull ghcr.io/lucky-aeon/mcp-gateway:latest
```

2. 自行构建 Docker 镜像

```bash
docker build -t mcp-gateway .
```

## 使用

运行 GitHub Docker 容器

```bash
docker run -d --name mcp-gateway -p 8080:8080 ghcr.io/lucky-aeon/mcp-gateway
```

运行自行构建的 Docker 容器

```bash
docker run -d --name mcp-gateway -p 8080:8080 mcp-gateway
```

## API

### 部署

支持：uvx、npx 或 sse url
```http
POST /deploy HTTP/1.1
Host: localhost:8080
Content-Type: application/json

{
    "mcpServers": {
        "time": {
            "url": "http://mcp-server:8080",  // url 和 command 二选一
            "command": "uvx",  // url 和 command 二选一
            "args": ["mcp-server-time", "--local-timezone=America/New_York"],  // 可选，command 的参数
            "env": {  // 可选，环境变量
                "KEY1": "VALUE1",
                "KEY2": "VALUE2"
            }
        }
    }
}
```

### 使用 MCP

#### 获取 SSE

```http
GET /{mcp-server-name}/sse HTTP/1.1
Host: localhost:8080
```

#### 发送消息

```http
POST /{mcp-server-name}/message HTTP/1.1
Host: localhost:8080
Content-Type: application/json

{
    "method": "tools/call",
    "params": {
        "name": "get_current_time",
        "arguments": {
            "timezone": "Asia/Seoul"
        }
    },
    "jsonrpc": "2.0",
    "id": 2
}
```

### 使用网关

网关和直连 MCP 的区别在于，只需要与网关交互，网关会自动将请求转发到对应的 MCP 服务器。在 call 时，需要在 method 前面添加 `mcpServerName` 内容，标识该请求来自哪个 MCP 服务器。

#### 获取 SSE

```http
GET /sse HTTP/1.1
Host: localhost:8080
```

这里 sse 是整个网关下所有的 MCP 服务器的 SSE 流。

当客户端订阅 sse 时，网关会为每个 MCP 服务器创建一个 SSE 连接，并将所有 MCP 服务器的 SSE 流合并到一起。

在响应的所有 tools/call 的结果中，会在 method 前面添加 `mcpServerName` 内容，标识该结果来自哪个 MCP 服务器。

#### 发送消息

```http
POST /message HTTP/1.1
Host: localhost:8080
Content-Type: application/json

{
    "method": "tools/call",
    "params": {
        "name": "{mcp-server-name}-get_current_time",
        "arguments": {
            "timezone": "Asia/Seoul"
        }
    },
    "jsonrpc": "2.0",
    "id": 2
}
```

获取网关下所有工具

```http
POST /message HTTP/1.1
Host: localhost:8080
Content-Type: application/json

{
    "method": "tools/list",
    "jsonrpc": "2.0",
    "id": 1
}

# SSE 响应 message event

{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "name": "{mcpServerName}-get_current_time",
        "description": "Get current time in a specific timezones",
        "inputSchema": {
          "type": "object",
          "properties": {
            "timezone": {
              "type": "string",
              "description": "IANA timezone name (e.g., 'America/New_York', 'Europe/London'). Use 'America/New_York' as local timezone if no timezone provided by the user."
            }
          },
          "required": [
            "timezone"
          ]
        }
      },
      {
        "name": "{mcpServerName}-convert_time",
        "description": "Convert time between timezones",
        "inputSchema": {
          "type": "object",
          "properties": {
            "source_timezone": {
              "type": "string",
              "description": "Source IANA timezone name (e.g., 'America/New_York', 'Europe/London'). Use 'America/New_York' as local timezone if no source timezone provided by the user."
            },
            "time": {
              "type": "string",
              "description": "Time to convert in 24-hour format (HH:MM)"
            },
            "target_timezone": {
              "type": "string",
              "description": "Target IANA timezone name (e.g., 'Asia/Tokyo', 'America/San_Francisco'). Use 'America/New_York' as local timezone if no target timezone provided by the user."
            }
          },
          "required": [
            "source_timezone",
            "time",
            "target_timezone"
          ]
        }
      }
    ]
  }
}
```