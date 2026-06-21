package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run cmd/gendoc/main.go <output-dir>")
		fmt.Println("  生成完整的 HTML 文档站到指定目录")
		os.Exit(1)
	}
	outDir := os.Args[1]
	os.MkdirAll(outDir, 0755)

	// Generate index.html
	indexHTML := generateIndex()
	os.WriteFile(outDir+"/index.html", []byte(indexHTML), 0644)

	// Generate API docs HTML
	apiHTML := generateAPIDocs()
	os.WriteFile(outDir+"/api.html", []byte(apiHTML), 0644)

	// Generate architecture HTML
	archHTML := generateArchitecture()
	os.WriteFile(outDir+"/architecture.html", []byte(archHTML), 0644)

	// Generate getting started HTML
	startHTML := generateGettingStarted()
	os.WriteFile(outDir+"/getting-started.html", []byte(startHTML), 0644)

	fmt.Printf("文档已生成到 %s/\n", outDir)
	fmt.Println("  index.html          - 首页")
	fmt.Println("  api.html            - API 接口文档")
	fmt.Println("  architecture.html   - 架构设计")
	fmt.Println("  getting-started.html - 快速开始")
}

func generateIndex() string {
	return page("LAN Agent Bus - 文档", `
<div class="hero">
  <h1>LAN Agent Bus</h1>
  <p class="subtitle">局域网 AI 智能体通信总线</p>
  <p class="desc">为 AI Agent 打造的去中心化 MCP 通信服务</p>
  <div class="badges">
    <span class="badge">Go</span>
    <span class="badge">TypeScript</span>
    <span class="badge">WebSocket</span>
    <span class="badge">mDNS</span>
    <span class="badge">MIT License</span>
  </div>
</div>

<div class="features">
  <div class="feature">
    <h3>零中心</h3>
    <p>无中心服务器，节点平等。Relay 可选，仅用于跨子网场景。</p>
  </div>
  <div class="feature">
    <h3>按需连接</h3>
    <p>默认零连接开销，mDNS 只维护在线列表。AI 决定何时建连。</p>
  </div>
  <div class="feature">
    <h3>AI 驱动</h3>
    <p>所有协作逻辑由 AI 之间通过文本自由推演，服务器不干预。</p>
  </div>
  <div class="feature">
    <h3>大文件传输</h3>
    <p>自动分块 64KB，4 并发传输，支持文件夹增量同步。</p>
  </div>
</div>

<div class="quick-links">
  <h2>快速开始</h2>
  <div class="code-block">
    <pre><code># 编译
go build -o lan-agent ./cmd/lan-agent
go build -o lan-server ./cmd/lan-server

# 运行 Agent A
./lan-agent -id agent-a -port 19100

# 运行 Agent B（同一局域网）
./lan-agent -id agent-b -port 19101

# 运行 Relay（可选，跨子网场景）
./lan-server -addr :19200</code></pre>
  </div>
</div>

<div class="quick-links">
  <h2>文档导航</h2>
  <a href="getting-started.html" class="nav-card">
    <h3>快速开始</h3>
    <p>安装、编译、运行</p>
  </a>
  <a href="api.html" class="nav-card">
    <h3>API 接口文档</h3>
    <p>11 个 MCP 工具完整定义</p>
  </a>
  <a href="architecture.html" class="nav-card">
    <h3>架构设计</h3>
    <p>系统架构、模块关系、数据流</p>
  </a>
</div>
`)
}

func generateAPIDocs() string {
	return page("API 接口文档 - LAN Agent Bus", `
<h1>MCP 工具接口文档</h1>
<p>所有工具通过 JSON-RPC 2.0 over WebSocket 调用。</p>

<h2>连接端点</h2>
<div class="endpoint">
  <span class="method">WS</span>
  <code>ws://&lt;agent-ip&gt;:&lt;port&gt;/ws</code>
</div>

<h2>工具列表</h2>

<div class="tool-group">
  <h3>🔍 发现与连接</h3>

  <div class="tool">
    <h4><code>lan_get_online_agents</code></h4>
    <p>获取所有在线 Agent（合并 mDNS + Relay）</p>
    <div class="params">
      <strong>输入：</strong> 无参数
    </div>
    <div class="response">
      <strong>返回：</strong>
      <pre><code>{"agents": ["agent-a", "agent-b"], "count": 2}</code></pre>
    </div>
  </div>

  <div class="tool">
    <h4><code>lan_open_connection</code></h4>
    <p>打开到指定 Agent 的 WebSocket 连接。<strong>发送消息或文件前必须先调用。</strong></p>
    <div class="params">
      <strong>输入：</strong>
      <table>
        <tr><th>参数</th><th>类型</th><th>必填</th><th>说明</th></tr>
        <tr><td><code>peer_id</code></td><td>string</td><td>✅</td><td>要连接的 Agent ID</td></tr>
      </table>
    </div>
    <div class="response">
      <strong>返回：</strong>
      <pre><code>{"status": "connected", "peer_id": "agent-b"}</code></pre>
    </div>
  </div>

  <div class="tool">
    <h4><code>lan_close_connection</code></h4>
    <p>关闭到指定 Agent 的连接</p>
    <div class="params">
      <strong>输入：</strong>
      <table>
        <tr><th>参数</th><th>类型</th><th>必填</th><th>说明</th></tr>
        <tr><td><code>peer_id</code></td><td>string</td><td>✅</td><td>要断开的 Agent ID</td></tr>
      </table>
    </div>
    <div class="response">
      <strong>返回：</strong>
      <pre><code>{"status": "disconnected", "peer_id": "agent-b"}</code></pre>
    </div>
  </div>
</div>

<div class="tool-group">
  <h3>💬 频道通信</h3>

  <div class="tool">
    <h4><code>lan_create_channel</code></h4>
    <p>创建通信频道并邀请 peer 加入</p>
    <div class="params">
      <strong>输入：</strong>
      <table>
        <tr><th>参数</th><th>类型</th><th>必填</th><th>说明</th></tr>
        <tr><td><code>channel_name</code></td><td>string</td><td>✅</td><td>频道名称</td></tr>
        <tr><td><code>peer_ids</code></td><td>string[]</td><td>✅</td><td>要邀请的 Agent ID 列表</td></tr>
      </table>
    </div>
    <div class="response">
      <strong>返回：</strong>
      <pre><code>{"channel_id": "ch-1234-xxx", "channel_name": "前端协作", "members": ["agent-a", "agent-b"]}</code></pre>
    </div>
  </div>

  <div class="tool">
    <h4><code>lan_leave_channel</code></h4>
    <p>退出通信频道</p>
    <div class="params">
      <strong>输入：</strong>
      <table>
        <tr><th>参数</th><th>类型</th><th>必填</th><th>说明</th></tr>
        <tr><td><code>channel_id</code></td><td>string</td><td>✅</td><td>频道 ID</td></tr>
      </table>
    </div>
    <div class="response">
      <strong>返回：</strong>
      <pre><code>{"status": "left", "channel_id": "ch-1234-xxx"}</code></pre>
    </div>
  </div>

  <div class="tool">
    <h4><code>lan_send_message</code></h4>
    <p>向频道发送消息。频道内所有成员的连接必须已打开。</p>
    <div class="params">
      <strong>输入：</strong>
      <table>
        <tr><th>参数</th><th>类型</th><th>必填</th><th>说明</th></tr>
        <tr><td><code>channel_id</code></td><td>string</td><td>✅</td><td>频道 ID</td></tr>
        <tr><td><code>message_body</code></td><td>string</td><td>✅</td><td>消息内容（文本/JSON）</td></tr>
      </table>
    </div>
    <div class="response">
      <strong>返回：</strong>
      <pre><code>{"status": "sent", "channel_id": "ch-1234-xxx", "recipients": 1}</code></pre>
    </div>
  </div>
</div>

<div class="tool-group">
  <h3>📁 文件服务</h3>

  <div class="tool">
    <h4><code>lan_share_file</code></h4>
    <p>分享本地文件到频道。支持大文件（自动分块 64KB，4 并发传输）。</p>
    <div class="params">
      <strong>输入：</strong>
      <table>
        <tr><th>参数</th><th>类型</th><th>必填</th><th>说明</th></tr>
        <tr><td><code>channel_id</code></td><td>string</td><td>✅</td><td>频道 ID</td></tr>
        <tr><td><code>file_path</code></td><td>string</td><td>✅</td><td>本地文件路径</td></tr>
      </table>
    </div>
    <div class="response">
      <strong>返回：</strong>
      <pre><code>{"status": "shared", "channel_id": "ch-xxx"}</code></pre>
    </div>
  </div>

  <div class="tool">
    <h4><code>lan_share_folder</code></h4>
    <p>分享整个文件夹。自动检测变化，只传输修改过的文件（增量同步）。</p>
    <div class="params">
      <strong>输入：</strong>
      <table>
        <tr><th>参数</th><th>类型</th><th>必填</th><th>说明</th></tr>
        <tr><td><code>channel_id</code></td><td>string</td><td>✅</td><td>频道 ID</td></tr>
        <tr><td><code>folder_path</code></td><td>string</td><td>✅</td><td>本地文件夹路径</td></tr>
      </table>
    </div>
    <div class="response">
      <strong>返回：</strong>
      <pre><code>{"status": "sharing", "channel_id": "ch-xxx", "folder": "/path/to/folder"}</code></pre>
    </div>
  </div>

  <div class="tool">
    <h4><code>lan_sync_folder</code></h4>
    <p>与远程 peer 同步文件夹。扫描变化，只发送 diff。</p>
    <div class="params">
      <strong>输入：</strong>
      <table>
        <tr><th>参数</th><th>类型</th><th>必填</th><th>说明</th></tr>
        <tr><td><code>folder_path</code></td><td>string</td><td>✅</td><td>本地文件夹路径</td></tr>
        <tr><td><code>peer_id</code></td><td>string</td><td>✅</td><td>远程 Agent ID</td></tr>
      </table>
    </div>
    <div class="response">
      <strong>返回：</strong>
      <pre><code>{"status": "syncing", "folder": "/path/to/folder", "peer_id": "agent-b"}</code></pre>
    </div>
  </div>
</div>

<div class="tool-group">
  <h3>📊 传输管理</h3>

  <div class="tool">
    <h4><code>lan_get_transfer_status</code></h4>
    <p>查询文件传输状态和进度</p>
    <div class="params">
      <strong>输入：</strong>
      <table>
        <tr><th>参数</th><th>类型</th><th>必填</th><th>说明</th></tr>
        <tr><td><code>transfer_id</code></td><td>string</td><td>✅</td><td>传输 ID</td></tr>
      </table>
    </div>
    <div class="response">
      <strong>返回：</strong>
      <pre><code>{
  "id": "tr-xxx",
  "file_path": "/path/to/file",
  "file_size": 1048576,
  "total_chunks": 16,
  "progress": 75.5,
  "status": "running"
}</code></pre>
    </div>
  </div>

  <div class="tool">
    <h4><code>lan_list_transfers</code></h4>
    <p>列出所有活跃的文件传输</p>
    <div class="params">
      <strong>输入：</strong> 无参数
    </div>
    <div class="response">
      <strong>返回：</strong>
      <pre><code>{
  "transfers": [...],
  "count": 3
}</code></pre>
    </div>
  </div>
</div>
`)
}

func generateArchitecture() string {
	return page("架构设计 - LAN Agent Bus", `
<h1>架构设计</h1>

<h2>系统架构</h2>
<div class="diagram">
<pre>
┌─────────────────────────────────────────────────────┐
│                    AI 决策层                          │
│  "我需要和 agent-b 协作，先看看谁在线"                    │
├─────────────────────────────────────────────────────┤
│                   MCP 工具层 (stdio)                  │
│  lan_get_online_agents / lan_open_connection / ...   │
├──────────────┬──────────────────┬───────────────────┤
│  P2P 模块     │   文件服务        │   频道管理          │
│  (WebSocket) │  (分块/并发/sync) │  (纯内存/无状态)     │
├──────────────┴──────────────────┴───────────────────┤
│              mDNS 发现 + Relay 中转 (可选)             │
└─────────────────────────────────────────────────────┘
</pre>
</div>

<h2>连接模型</h2>
<div class="diagram">
<pre>
默认状态：零连接
  Agent A ──── mDNS ──── Agent B    (只感知在线，不建连)

AI 决定交互：
  Agent A ──WebSocket──> Agent B    (按需建连，用完可关)

跨子网场景：
  Agent A ──WS──> Relay ──WS──> Agent B
</pre>
</div>

<h2>模块关系</h2>
<table class="arch-table">
  <tr><th>模块</th><th>职责</th><th>关键特性</th></tr>
  <tr><td><code>protocol</code></td><td>WebSocket 消息协议</td><td>JSON 消息、连接封装</td></tr>
  <tr><td><code>p2p</code></td><td>P2P 传输 + mDNS 发现</td><td>按需连接、在线列表、WebSocket 服务端</td></tr>
  <tr><td><code>channel</code></td><td>频道管理</td><td>纯内存、无状态、断线即散</td></tr>
  <tr><td><code>fileservice</code></td><td>文件服务</td><td>分块传输、并发、重试、文件夹同步</td></tr>
  <tr><td><code>filetransfer</code></td><td>文件传输基础</td><td>文件切片、SHA-256 校验</td></tr>
  <tr><td><code>plugins</code></td><td>插件系统</td><td>事件驱动、Hook、过滤、转换</td></tr>
  <tr><td><code>logger</code></td><td>结构化日志</td><td>slog、JSON 输出、组件标签</td></tr>
  <tr><td><code>mcp</code></td><td>MCP JSON-RPC 服务</td><td>11 个工具、stdio 接口</td></tr>
  <tr><td><code>relay</code></td><td>Relay 客户端</td><td>跨子网中转、在线列表查询</td></tr>
</table>

<h2>数据流</h2>

<h3>消息发送</h3>
<div class="diagram">
<pre>
AI 调用 lan_send_message
  → MCP Server 解析参数
  → Agent 查找频道成员
  → 对每个成员：
      有连接 → 直接 WebSocket 发送
      无连接 → 返回错误 "call lan_open_connection first"
</pre>
</div>

<h3>文件传输</h3>
<div class="diagram">
<pre>
AI 调用 lan_share_file
  → FileService.SplitFile() 分块 (64KB)
  → 创建 Transfer 对象
  → 4 个 goroutine 并发发送
  → 每个 chunk: ReadChunk → WebSocket 发送
  → 全部完成 → Status=completed
  → 插件触发 EventTransferDone
</pre>
</div>

<h3>文件夹同步</h3>
<div class="diagram">
<pre>
AI 调用 lan_sync_folder
  → FolderSync.ScanFolder() 扫描当前状态
  → FolderSync.LoadManifest() 加载上次状态
  → DiffFolders() 计算 diff (adds/modifies/deletes)
  → 对 adds + modifies: FileService.SendFile()
  → 保存新 manifest 到 .lan-sync-manifest.json
</pre>
</div>

<h2>协议规范</h2>
<p>详见 <a href="protocol.html">协议规范文档</a></p>

<h3>WebSocket 消息格式</h3>
<div class="code-block">
<pre><code>{
  "type": "text",
  "id": "1718971234-abc123",
  "from": "agent-a",
  "channel_id": "ch-1234",
  "content": "你好",
  "ts": 1718971234
}</code></pre>
</div>

<h3>握手流程</h3>
<div class="diagram">
<pre>
发起方                        接收方
  |--- register ──────────->|
  |<------ register_ok -----|
  |                          |
  |   连接建立，可以通信       |
</pre>
</div>
`)
}

func generateGettingStarted() string {
	return page("快速开始 - LAN Agent Bus", `
<h1>快速开始</h1>

<h2>环境要求</h2>
<ul>
  <li>Go 1.21+</li>
  <li>Node.js 18+ (TypeScript 客户端)</li>
  <li>同一局域网 (P2P 发现) 或 Relay 服务器 (跨子网)</li>
</ul>

<h2>编译</h2>
<div class="code-block">
<pre><code># 克隆项目
git clone https://github.com/user/lan-a2a.git
cd lan-agent-bus

# 编译 Go 客户端和服务端
go build -o lan-agent ./cmd/lan-agent
go build -o lan-server ./cmd/lan-server</code></pre>
</div>

<h2>运行</h2>

<h3>场景 1：同一局域网</h3>
<div class="code-block">
<pre><code># 终端 1：启动 Agent A
./lan-agent -id agent-a -port 19100

# 终端 2：启动 Agent B
./lan-agent -id agent-b -port 19101

# 几秒后自动发现：
# [online] agent-b (lan)</code></pre>
</div>

<h3>场景 2：跨子网（需要 Relay）</h3>
<div class="code-block">
<pre><code># 服务器上启动 Relay
./lan-server -addr :19200

# 不同子网的 Agent 启动后自动发现 Relay
./lan-agent -id agent-c
# [discovery] found relay at 192.168.1.100:19200
# [relay] connected to 192.168.1.100:19200</code></pre>
</div>

<h3>场景 3：TypeScript 客户端</h3>
<div class="code-block">
<pre><code>cd clients/typescript
npm install
npm run dev -- --id agent-d --port 19103</code></pre>
</div>

<h2>MCP 配置</h2>
<p>在你的 AI 工具（Cline/Cursor 等）中配置 MCP：</p>
<div class="code-block">
<pre><code>{
  "mcpServers": {
    "lan-agent": {
      "command": "/path/to/lan-agent",
      "args": ["-id", "my-agent"]
    }
  }
}</code></pre>
</div>

<h2>验证</h2>
<div class="code-block">
<pre><code># 在 AI 工具中调用：
lan_get_online_agents
# → {"agents": ["agent-b", "agent-c"], "count": 2}

lan_open_connection(peer_id="agent-b")
# → {"status": "connected", "peer_id": "agent-b"}

lan_create_channel(channel_name="测试", peer_ids=["agent-b"])
# → {"channel_id": "ch-xxx", "members": ["my-agent", "agent-b"]}

lan_send_message(channel_id="ch-xxx", message_body="Hello!")
# → {"status": "sent", "recipients": 1}</code></pre>
</div>

<h2>Docker 部署</h2>
<div class="code-block">
<pre><code># 构建 Relay 镜像
docker build -t lan-server .

# 运行
docker run -d \\
  --name lan-relay \\
  -p 19200:19200 \\
  -p 19201:19201 \\
  lan-server

# 查看状态
curl http://localhost:19201/status</code></pre>
</div>
`)
}

func page(title, body string) string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + title + `</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;line-height:1.6;color:#e0e0e0;background:#0d1117;padding:0}
.container{max-width:960px;margin:0 auto;padding:2rem 1.5rem}
h1{font-size:2rem;margin-bottom:1rem;color:#58a6ff}
h2{font-size:1.4rem;margin:2rem 0 1rem;color:#c9d1d9;border-bottom:1px solid #30363d;padding-bottom:0.5rem}
h3{font-size:1.15rem;margin:1.5rem 0 0.75rem;color:#c9d1d9}
h4{font-size:1rem;margin:0.5rem 0;color:#58a6ff}
p{margin:0.5rem 0;color:#8b949e}
a{color:#58a6ff;text-decoration:none}
a:hover{text-decoration:underline}
code{background:#161b22;padding:0.15em 0.4em;border-radius:4px;font-size:0.9em;color:#79c0ff}
pre{background:#161b22;border:1px solid #30363d;border-radius:8px;padding:1rem;overflow-x:auto;font-size:0.85rem;line-height:1.5}
pre code{background:none;padding:0;color:#c9d1d9}
table{width:100%;border-collapse:collapse;margin:1rem 0}
th,td{padding:0.6rem 1rem;text-align:left;border-bottom:1px solid #30363d}
th{color:#c9d1d9;font-weight:600;background:#161b22}
td{color:#8b949e}
.hero{text-align:center;padding:3rem 0}
.hero h1{font-size:3rem;color:#58a6ff;margin-bottom:0.5rem}
.subtitle{font-size:1.3rem;color:#c9d1d9}
.desc{color:#8b949e;margin:1rem 0}
.badges{display:flex;gap:0.5rem;justify-content:center;margin-top:1rem}
.badge{background:#1f6feb33;color:#58a6ff;padding:0.25rem 0.75rem;border-radius:999px;font-size:0.85rem}
.features{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:1rem;margin:2rem 0}
.feature{background:#161b22;border:1px solid #30363d;border-radius:8px;padding:1.25rem}
.feature h3{color:#58a6ff;margin-bottom:0.5rem}
.feature p{font-size:0.9rem}
.nav-card{display:block;background:#161b22;border:1px solid #30363d;border-radius:8px;padding:1.25rem;margin:0.75rem 0;transition:border-color 0.2s}
.nav-card:hover{border-color:#58a6ff;text-decoration:none}
.nav-card h3{color:#58a6ff;margin-bottom:0.25rem}
.nav-card p{margin:0;font-size:0.9rem}
.tool-group{margin:1.5rem 0}
.tool{background:#161b22;border:1px solid #30363d;border-radius:8px;padding:1.25rem;margin:1rem 0}
.tool h4{margin-top:0}
.params{margin:0.75rem 0}
.response{margin-top:0.75rem}
.endpoint{background:#161b22;border:1px solid #30363d;border-radius:8px;padding:1rem;margin:1rem 0;display:flex;align-items:center;gap:1rem}
.method{background:#238636;color:#fff;padding:0.2rem 0.6rem;border-radius:4px;font-weight:700;font-size:0.85rem}
.diagram{background:#161b22;border:1px solid #30363d;border-radius:8px;padding:1rem;margin:1rem 0;overflow-x:auto}
.diagram pre{border:none;background:none;margin:0}
.arch-table{margin:1rem 0}
.code-block{margin:1rem 0}
ul{margin:0.5rem 0 0.5rem 1.5rem}
li{margin:0.25rem 0;color:#8b949e}
footer{text-align:center;padding:2rem 0;color:#484f58;font-size:0.85rem;border-top:1px solid #30363d;margin-top:3rem}
</style>
</head>
<body>
<div class="container">
<nav style="margin-bottom:2rem;font-size:0.9rem">
  <a href="index.html">首页</a> &middot;
  <a href="getting-started.html">快速开始</a> &middot;
  <a href="api.html">API 文档</a> &middot;
  <a href="architecture.html">架构设计</a>
</nav>
` + body + `
<footer>LAN Agent Bus v0.1.0 &middot; MIT License</footer>
</div>
</body>
</html>`
}

// suppress unused import
var _ = strings.TrimSpace
