# lumidive 開発・協業規約 (AGENTS.md)

本ドキュメントは、AIエージェントおよび開発者が `lumidive` リポジトリで作業する際の共通規約を定義します。

---

## 0. リポジトリ概要

### プロジェクト情報

| 項目 | 内容 |
|:---|:---|
| リポジトリ | `github.com/AobaIwaki123/lumidive` |
| 言語 | Go 1.23+ |
| ライセンス | MIT |
| 目的 | TicketDive イベント情報・リアルタイム券種在庫・アンケート設問の取得・標準化、iCalendar (.ics) 生成、OpenAPI 準拠 REST API サーバー |

### ディレクトリ構成

```
lumidive/
├── api/                  # OpenAPI 仕様 (openapi.yaml) と oapi-codegen 設定
├── cmd/lumidive/         # エントリポイント (main.go)
├── pkg/
│   ├── api/              # oapi-codegen 自動生成コード (lumidive.gen.go)
│   ├── server/           # HTTP ハンドラ / REST API サーバー
│   └── ticketdive/       # TicketDive クライアント、SSRパーサー、TTLキャッシュ、iCal生成
├── scripts/              # 開発・検証用スクリプト (verify-all.sh)
├── tools/                # 開発ツール定義 (tools.go)
└── Dockerfile            # マルチステージ Distroless コンテナ
```

---

## 1. 開発フローおよびPR運用ルール

- **main への直接 Push の絶対禁止**:
  - すべての変更は必ずトピックブランチを切って Pull Request 経由で反映します。
- **ローカル完全検証の義務付け**:
  - コミット・Push 前に必ず `./scripts/verify-all.sh` を実行し、全チェック（Code Generation Drift Check、golangci-lint、go test -race、Build）が 100% 成功することを確認します。
- **PR の単一責務の原則 (Single Responsibility PR)**:
  - 1 PR = 1 つの明確な責務に限定します。
- **ドキュメントにおける絵文字の不使用**:
  - 公式ドキュメントでは原則として絵文字を使用しません。
- **エージェントによる自律的マージの禁止**:
  - マージ判断およびリリース判定はユーザーが行います。
