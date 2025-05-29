# Slack All Contexts

Slackの指定したチャンネルのすべてのメッセージとスレッドをSQLiteデータベースに保存し、テキストファイルとして出力するGoプログラムです。

## 機能

- 指定したSlackチャンネルのすべてのメッセージを取得
- スレッドの返信も含めて関連付けで保存
- SQLiteデータベースへの永続化
- Slack APIのレート制限を考慮した処理
- 増分更新対応（既に取得したメッセージはスキップ）
- データベースからのテキスト形式でのエクスポート機能
- 全チャンネル一括エクスポート機能

## セットアップ

### 1. 依存関係のインストール

```bash
go mod tidy
```

### 2. Slack Botトークンの取得

1. [Slack API](https://api.slack.com/apps)でアプリを作成
2. Bot Token Scopesで以下の権限を追加：
   - `channels:history`
   - `channels:read`
   - `groups:history`（プライベートチャンネルの場合）
   - `groups:read`（プライベートチャンネルの場合）
3. アプリをワークスペースにインストール
4. Bot User OAuth Tokenを取得（`xoxb-`で始まるトークン）

## 使用方法

### ビルド

```bash
go build -o slack-all-contexts
```

### メッセージの取得（fetchモード）

```bash
# 環境変数でトークンを設定
export SLACK_BOT_TOKEN="xoxb-your-bot-token"
./slack-all-contexts -channel C1234567890

# または直接トークンを指定
./slack-all-contexts -token xoxb-your-bot-token -channel C1234567890

# データベースファイル名を指定
./slack-all-contexts -channel C1234567890 -db my_slack_data.db
```

### データのエクスポート（exportモード）

```bash
# 特定のチャンネルをテキストファイルにエクスポート
./slack-all-contexts -mode export -channel C1234567890 -output channel_export.txt

# 全チャンネルを指定ディレクトリにエクスポート
./slack-all-contexts -mode export -output-dir ./exports

# 既存のデータベースファイルを指定してエクスポート
./slack-all-contexts -mode export -db my_slack_data.db -channel C1234567890 -output my_export.txt
```

### チャンネルIDの取得方法

1. Slackでチャンネルを右クリック
2. 「リンクをコピー」を選択
3. URLの最後の部分がチャンネルID（例：`C1234567890`）

## データベース構造

### channels テーブル
- `id`: チャンネルID
- `name`: チャンネル名
- `created_at`: レコード作成日時

### messages テーブル
- `ts`: メッセージのタイムスタンプ（主キー）
- `channel_id`: チャンネルID
- `user_id`: 投稿者のユーザーID
- `text`: メッセージテキスト
- `thread_ts`: スレッドのタイムスタンプ（親メッセージの場合）
- `reply_count`: 返信数
- `created_at`: レコード作成日時

### replies テーブル
- `ts`: 返信のタイムスタンプ（主キー）
- `thread_ts`: 親メッセージのタイムスタンプ
- `channel_id`: チャンネルID
- `user_id`: 返信者のユーザーID
- `text`: 返信テキスト
- `created_at`: レコード作成日時

## レート制限対応

Slack APIのレート制限（1秒あたり1リクエスト）を考慮し、`golang.org/x/time/rate`パッケージを使用してリクエスト間隔を制御しています。

## エクスポートファイル形式

エクスポートされるテキストファイルは以下の形式で出力されます：

```
# Slack Channel Export: #general
Channel ID: C1234567890
Export Date: 2024-01-01 12:00:00
Total Messages: 150

======================================================================

[2024-01-01 10:30:45] user123:
こんにちは！新しいプロジェクトについて話し合いましょう。

  Thread Replies (2):
  [2024-01-01 10:31:20] user456: いいですね！どんなプロジェクトですか？
  [2024-01-01 10:32:15] user123: WebアプリケーションのRESTful APIを作る予定です。

--------------------------------------------------------------------------------
```

## 注意事項

- 大量のメッセージがあるチャンネルでは処理に時間がかかります
- Botがチャンネルにアクセスできる権限が必要です
- プライベートチャンネルの場合は追加の権限設定が必要です
- エクスポート機能はSlackトークンなしで使用可能です（既存のデータベースから出力）