# Git Worktree CLI Tool "gw" 実装依頼

## 概要
Gitのworktreeをより扱いやすくするCLIツール「gw」をGoで実装してください。

## 技術スタック
- **言語**: Go
- **CLIフレームワーク**: cobra
- **インタラクティブUI**: bubbletea + lipgloss  
- **Git操作**: go-git または git コマンド実行
- **ビルドツール**: Go modules

## 機能要件

### 1. `gw start xxx [base branch]` コマンド
```bash
gw start 123 main  # issue #123のworktreeを作成、mainブランチベース
gw start 456       # base branchのデフォルトはmain
```

**動作**:
1. `git worktree add ../{repository.name}-{xxx} -b {xxx}/impl [base branch]` を実行
2. 作成されたworktreeディレクトリに移動
3. プロジェクトの種類を自動判定し、適切なセットアップコマンドを実行:
   - `package.json` → `npm install` / `pnpm install` / `yarn install`
   - `Cargo.toml` → `cargo build`
   - `go.mod` → `go mod download`
   - `requirements.txt` → `pip install -r requirements.txt`
   - `Gemfile` → `bundle install`

### 2. `gw end xxx` コマンド  
```bash
gw end 123  # issue #123のworktreeを削除
gw end      # インタラクティブ選択
```

**動作**:
1. 指定されたworktreeの安全性チェック:
   - uncommittedな変更がないか
   - unpushedなコミットがないか  
   - originにマージされていないか
2. 警告がある場合は確認プロンプトを表示
3. `git worktree remove ../{repository.name}-{xxx}` を実行

### 3. インタラクティブUI (xxxが未指定の場合)
- `git worktree list` の結果を表示
- j/kキーで上下移動
- Enterで選択、qで終了
- 選択されたworktreeに対してendコマンドを実行

## ディレクトリ構造
```
gw/
├── main.go
├── go.mod
├── go.sum
├── cmd/
│   ├── root.go      # ルートコマンド設定
│   ├── start.go     # startコマンド実装
│   └── end.go       # endコマンド実装
├── internal/
│   ├── git/
│   │   ├── worktree.go    # worktree操作
│   │   ├── status.go      # ステータスチェック
│   │   └── repository.go  # リポジトリ情報取得
│   ├── detect/
│   │   └── package.go     # パッケージマネージャー検出
│   ├── ui/
│   │   └── selector.go    # インタラクティブ選択UI
│   └── config/
│       └── config.go      # 設定管理
└── README.md
```

## 実装の詳細要件

### エラーハンドリング
- 各Git操作でのエラーを適切にキャッチ
- ユーザーフレンドリーなエラーメッセージ
- 操作の途中で失敗した場合のロールバック

### 設定
- デフォルトのbase branchを設定可能 (`.gwconfig` など)
- worktreeの作成場所を設定可能

### ログ・出力
- 実行中の操作を分かりやすく表示
- デバッグモード (`--verbose` フラグ)
- カラー出力対応

### バリデーション
- 既存のworktreeとの重複チェック
- 無効なissue番号の検証
- Git リポジトリ内での実行チェック

## パッケージ依存関係
```go
module gw

go 1.21

require (
    github.com/spf13/cobra v1.8.0
    github.com/charmbracelet/bubbletea v0.25.0
    github.com/charmbracelet/lipgloss v0.9.1
    github.com/go-git/go-git/v5 v5.11.0
)
```

## 実装時の注意点
1. **クロスプラットフォーム対応**: Windows/Mac/Linuxで動作すること
2. **Git操作の安全性**: 既存のworktreeやブランチを破壊しないこと  
3. **ユーザビリティ**: 直感的で分かりやすいUI/UX
4. **パフォーマンス**: 大きなリポジトリでも快適に動作すること
5. **テスタビリティ**: 単体テストを書きやすい構造にすること

## 成果物
- 完全に動作するCLIツールのソースコード
- ビルド・インストール手順
- 使用方法の簡潔なドキュメント
- 主要機能の単体テスト

このプロンプトに基づいて、実用的で保守性の高いCLIツールを実装してください。