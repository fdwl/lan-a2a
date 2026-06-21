# LanA2A

[English](../README.md) | [中文](README.zh-CN.md) | [日本語](README.ja.md)

**A2A プロトコルの LAN 実装** — [Agent2Agent (A2A)](https://a2a-protocol.org) 標準プロトコルベースの、AI Agent 用分散型 LAN 通信サービス。

> [Xiaomi MiMo](https://github.com/XiaomiMiMo) チームによって開発。Powered by MiMo AI.

## 主な機能

- **標準プロトコル**: A2A Go SDK ベース、既製品を活用
- **LAN 最適化**: mDNS ディスカバリー + WebSocket トランスポート（A2A はネイティブに HTTP）
- **オンデマンド接続**: デフォルトはゼロ接続、AI が接続タイミングを決定
- **デュアルモード**: P2P Lobby（サーバーなし）+ Relay（サーバーあり）
- **ファイル転送**: チャンク分割同時転送 + フォルダ增量同步
- **MCP インターフェース**: AI Agent が LAN 通信に使用する 11 のツール

## アーキテクチャ

```
┌─────────────────────────────────────────────────────┐
│                    A2A プロトコル層                    │
│  AgentCard / Task / Message / Skills                 │
├──────────────┬──────────────────┬───────────────────┤
│  LAN トランスポート │   ファイルサービス    │   チャンネル管理     │
│  mDNS + WS   │  (チャンク/同時/sync) │  (P2P/Relay)     │
├──────────────┴──────────────────┴───────────────────┤
│              A2A Go SDK (a2asrv / a2aclient)         │
└─────────────────────────────────────────────────────┘
```

## クイックスタート

```bash
# ビルド
make build

# Agent 実行
./lan-a2a -id agent-a -port 19100

# Relay 実行（オプション、サブネット跨ぎシナリオ用）
./lan-relay -addr :19200
```

## A2A 互換性

| A2A 概念 | LanA2A 実装 |
|----------|-------------|
| AgentCard | Profile + LAN ブロードキャスト |
| Agent Discovery | mDNS（LAN）+ Relay クエリ |
| Task Lifecycle | チャンネルベースのタスク追跡 |
| Message/Part | WebSocket JSON メッセージ |
| Transport | WebSocket（LAN 最適化） |
| Skills | Profile.Roles |

## MCP ツール

| ツール | 説明 |
|--------|------|
| `lan_get_online_agents` | オンライン Agent を取得（AgentCard 情報付き） |
| `lan_open_connection` | WebSocket 接続を開く |
| `lan_close_connection` | 接続を閉じる |
| `lan_create_channel` | チャンネル作成（P2P Lobby または Relay モード） |
| `lan_send_message` | メッセージ送信 |
| `lan_share_file` | ファイル共有（大ファイルは自動チャンク分割） |
| `lan_share_folder` | フォルダ共有（增量同步） |
| `lan_sync_folder` | リモート peer とフォルダを同期 |
| `lan_get_transfer_status` | 転送状態を取得 |
| `lan_list_transfers` | アクティブな転送を一覧表示 |

## プロジェクト構造

```
lan-a2a/
├── cmd/
│   ├── lan-a2a/           # Agent クライアント
│   ├── lan-relay/         # Relay サーバー
│   └── gendoc/            # ドキュメント生成器
├── internal/
│   ├── adapter/           # A2A プロトコルアダプター
│   ├── protocol/          # WebSocket メッセージプロトコル
│   ├── channel/           # チャンネル管理（P2P Lobby / Relay）
│   ├── fileservice/       # ファイルサービス（チャンク/同時/sync）
│   ├── filetransfer/      # 受信ファイル組立
│   ├── plugins/           # プラグインシステム
│   ├── profile/           # Agent 身元（→ A2A AgentCard）
│   ├── logger/            # 構造化ログ（slog）
│   ├── mcp/               # MCP JSON-RPC サーバー
│   ├── p2p/               # P2P トランスポート + mDNS
│   └── relay/             # Relay クライアント
├── clients/typescript/    # TypeScript クライアント
├── docs/
│   ├── openapi.yaml       # OpenAPI 仕様
│   └── protocol.md        # ワイヤプロトコル仕様
└── Dockerfile
```

## Docker

```bash
# Agent イメージをビルド
docker build --target agent -t lan-a2a:latest .

# Relay イメージをビルド
docker build --target relay -t lan-relay:latest .

# Relay を実行
docker run -d --name lan-relay -p 19200:19200 -p 19201:19201 lan-relay:latest
```

## 設定

### Agent

| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `-id` | `<hostname>-<pid>` | Agent ID |
| `-port` | `19100 + PID%1000` | WebSocket 待受ポート |

### Relay

| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `-addr` | `:19200` | WebSocket 待受アドレス |
| `-http` | `:19201` | HTTP ステータスアドレス |

## コントリビューション

[CONTRIBUTING.md](../CONTRIBUTING.md) を参照してください。

## セキュリティ

[SECURITY.md](../SECURITY.md) を参照してください。

## 変更履歴

[CHANGELOG.md](../CHANGELOG.md) を参照してください。

## ライセンス

MIT ライセンス。詳細は [LICENSE](../LICENSE) を参照。

---

<p align="center">
  <a href="https://github.com/XiaomiMiMo">Xiaomi MiMo</a> チームによって開発
</p>
