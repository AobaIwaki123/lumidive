# lumidive

TicketDive（チケットダイブ: ticketdive.com / t-dv.com）のイベント情報・チケット販売状況の解析プロキシ、CLIツール、およびカレンダー連携（iCalendar）サーバーです。

Web UIが内包されており、ブラウザからURLを入力するだけで、イベントの開演時間・会場・出演グループ・チケット残席を即座に確認し、GoogleカレンダーやAppleカレンダーへワンクリックで登録できます。

---

## 主な特徴

- **誰でも使える Web UI**: ブラウザでアクセスし、TicketDiveのイベントURLまたはIDを入れるだけで、イベント詳細の閲覧とカレンダー連携URL（`.ics`）の発行が可能。
- **各種URLの自動判別**:
  - 通常URL: `https://ticketdive.com/event/<id>`
  - 公式短縮URL: `https://t-dv.com/<id>`
  - FC会員先行URL: `https://ticketdive.com/event/fc/<id>`
  - イベントID直指定: `plkt1022` 等
- **リッチなメタデータ抽出**: Next.jsのSSRデータ（`__NEXT_DATA__`）から、イベント概要、タイムテーブル、会場、出演者、全券種名、チケット定価・手数料、販売状況（販売中/完売/残りわずか）、アンケート選択肢などを構造化。
- **カレンダー連携（iCalendar / RFC 5545）**: iPhone/MacのカレンダーアプリやGoogleカレンダーに登録可能な`.ics`形式を即時生成。
- **高スループット & 多重リクエスト防止**: インメモリTTLキャッシュと `singleflight` による重複取得の合流処理を内包。
- **軽量 & 安全**: 外部Node/フロントエンドビルド不要（Goの`embed`による単一バイナリ）、Distrolessコンテナ（約9MB）。

---

## 使い方

### 1. Webブラウザで使う（推奨）

サーバー（[https://lumidive.aooba.net/](https://lumidive.aooba.net/)）を起動してブラウザでアクセスします。

1. 入力欄に TicketDive のイベントURL（またはID）を貼り付け、「情報を取得」をクリック。
2. イベント名、日時、会場、出演者、各チケットの在庫状況がプレビューされます。
3. **カレンダー連携**:
   - 「URLコピー」をクリックし、iPhoneのカレンダー「新規カレンダー購読」やGoogleカレンダーの「URLで追加」に貼り付けると、予定がカレンダーに自動登録されます。
   - 「ダウンロード」をクリックして直接 `.ics` ファイルを保存することも可能です。

---

### 2. コマンドライン（CLI）で使う

```bash
# イベント情報を解析してJSON形式で標準出力
lumidive parse https://ticketdive.com/event/plkt1022
lumidive parse t-dv.com/plkt1022
lumidive parse plkt1022

# アーティスト情報＆出演予定イベント一覧を解析（JSON）
lumidive parse https://ticketdive.com/artist/yoruami

# iCalendar (.ics) データを標準出力（イベント単体 or アーティスト全公演）
lumidive ical plkt1022 > event.ics
lumidive ical https://ticketdive.com/artist/yoruami > artist.ics

# Web UI & API サーバーの起動
lumidive server --port 8080 --cache-ttl 60s
```

---

### 3. APIエンドポイント（開発者向け）

| メソッド | パス | 説明 |
|:---|:---|:---|
| `GET` | `/` | Web UI（ブラウザ向けダッシュボード） |
| `GET` | `/healthz` | ヘルスチェック |
| `GET` | `/api/v1/events/{eventId}` | イベントIDからメタデータを取得（JSON） |
| `GET` | `/api/v1/events?url={url}` | URLパラメータからメタデータを取得（JSON） |
| `POST` | `/api/v1/events/parse` | リクエストボディ（`{"url": "..."}`）からパース |
| `POST` | `/api/v1/events/batch` | 複数イベントの一括パース |
| `GET` | `/api/v1/events/{eventId}/ical` | iCalendar形式（.ics）でイベントを取得 |
| `GET` | `/api/v1/artists/{artistId}` | アーティスト情報＆出演予定イベント一覧を取得（JSON） |
| `GET` | `/api/v1/artists/{artistId}/ical` | アーティストの全出演予定イベントをiCal配信（.ics） |

#### レスポンス例 (`GET /api/v1/events/plkt1022`)
```json
{
  "apiVersion": "1.0",
  "success": true,
  "cached": true,
  "data": {
    "id": "drIPQl23ZVboh1BWQy6y",
    "name": "2026/10/22(木) 「笑う門にはプリュきたる!! 」@WOMBLIVE",
    "slug": "plkt1022",
    "url": "https://ticketdive.com/event/plkt1022",
    "stages": [
      {
        "openAt": "2026-10-22T06:15:00.000Z",
        "startAt": "2026-10-22T06:45:00.000Z",
        "venue": { "name": "WOMBLIVE" },
        "artists": [
          { "name": "ハニースパイスRe." },
          { "name": "JAPANARIZM" }
        ]
      }
    ],
    "ticketGroups": [
      {
        "name": "一般販売",
        "types": [
          {
            "name": "一般前方チケット",
            "price": 4200,
            "totalPrice": 4540,
            "isSoldOut": false
          }
        ]
      }
    ]
  }
}
```

---

## ビルド & 開発コマンド (Make)

プロジェクトの定型コマンドは `Makefile` に集約されています：

```bash
make help          # 利用可能なコマンド一覧を表示
make build         # バイナリを bin/lumidive にビルド
make run           # ローカルサーバーをポート 8080 で起動
make test          # -race フラグ付きユニット・インテグレーションテスト実行
make lint          # golangci-lint による静的解析
make verify        # フルローカル検証（スキーマドリフト・リント・テスト・ビルド）
make generate      # OpenAPI コード生成および仕様ファイルの同期
make docker-build  # ローカル Docker イメージのビルド
make clean         # ビルド成果物および一時ファイルのクリーンアップ
```

---

## ライセンス

MIT License
