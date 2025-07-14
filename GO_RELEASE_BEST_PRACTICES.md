# Go パッケージリリースのベストプラクティス

## 1. バージョニング

### セマンティックバージョニング
- **必ず `v` プレフィックスを使用**: `v1.0.0`, `v1.2.3`
- Go モジュールは `v` プレフィックスを要求します
- メジャーバージョン 2 以降は、モジュールパスに含める必要があります
  ```go
  module github.com/username/gw/v2
  ```

### バージョン管理の原則
- **v0.x.x**: 開発中、破壊的変更OK
- **v1.0.0**: 安定版リリース、後方互換性を保つ
- **破壊的変更**: メジャーバージョンを上げる

## 2. リリース前チェックリスト

### コード品質
```bash
# テスト実行
make test

# 静的解析
make lint

# ビルド確認（全プラットフォーム）
make build-all
```

### ドキュメント更新
- [ ] README.md の更新
- [ ] CHANGELOG.md の更新
- [ ] godoc コメントの確認
- [ ] 使用例の更新

## 3. GitHub リリースの自動化

### タグプッシュによる自動リリース
```bash
# タグを作成
git tag v0.1.0 -m "Release v0.1.0"

# タグをプッシュ（自動的にリリースが作成される）
git push origin v0.1.0
```

### GoReleaser の利点
- マルチプラットフォームビルド
- 自動的なチェンジログ生成
- Docker イメージの作成（設定により）
- checksums.txt の自動生成

## 4. Go モジュールのベストプラクティス

### go.mod の管理
```bash
# 依存関係の整理
go mod tidy

# 依存関係の更新
go get -u ./...

# 特定のバージョンに固定
go get github.com/spf13/cobra@v1.8.0
```

### プロキシとサムデータベース
- リリース後、`proxy.golang.org` に反映されるまで数分かかる
- 確認方法:
  ```bash
  curl https://proxy.golang.org/github.com/username/gw/@v/list
  ```

## 5. バイナリ配布

### 推奨される配布方法
1. **GitHub Releases**: 直接ダウンロード
2. **go install**: Go 開発者向け
3. **Docker**: コンテナ環境向け
4. **OS パッケージマネージャー**: apt, yum, etc.

### バイナリサイズ最適化
```go
// ldflags でバイナリサイズを削減
-ldflags="-s -w"
// -s: シンボルテーブルを削除
// -w: DWARF デバッグ情報を削除
```

## 6. 後方互換性

### 互換性を保つためのルール
1. **エクスポートされた関数のシグネチャを変更しない**
2. **構造体にフィールドを追加する場合は最後に追加**
3. **インターフェースにメソッドを追加しない**（v2 まで待つ）
4. **定数や変数の値を変更しない**

### 非推奨の管理
```go
// Deprecated: Use NewFunction instead. This will be removed in v2.0.0
func OldFunction() {
    // ...
}
```

## 7. セキュリティ

### リリース時のセキュリティ対策
1. **依存関係の脆弱性チェック**
   ```bash
   go list -json -m all | nancy sleuth
   ```

2. **バイナリの署名**（GoReleaser で設定可能）

3. **SBOM (Software Bill of Materials) の生成**

## 8. パフォーマンス

### リリース前のベンチマーク
```bash
# ベンチマークの実行
go test -bench=. -benchmem ./...

# 前バージョンとの比較
go test -bench=. -benchmem ./... > new.txt
benchcmp old.txt new.txt
```

## 9. ドキュメント

### pkg.go.dev の活用
- README.md は自動的に表示される
- Example テストは自動的にドキュメントに含まれる
- バージョンごとのドキュメントが保存される

### Example テストの書き方
```go
func ExampleNewFunction() {
    result := NewFunction()
    fmt.Println(result)
    // Output: expected output
}
```

## 10. 継続的な改善

### メトリクスの収集
- ダウンロード数の追跡（GitHub Insights）
- イシューとプルリクエストの管理
- ユーザーフィードバックの収集

### リリースサイクル
- 定期的なリリース（月1回など）
- セキュリティ修正は即座にリリース
- 機能追加は計画的にバンドル

## まとめ

成功するGoパッケージのリリースには：
1. **自動化**: CI/CD パイプラインの活用
2. **一貫性**: セマンティックバージョニングの遵守
3. **透明性**: 明確なチェンジログとドキュメント
4. **信頼性**: 十分なテストとレビュー
5. **アクセシビリティ**: 複数の配布チャネル

これらのベストプラクティスに従うことで、ユーザーにとって使いやすく、メンテナンスしやすいGoパッケージをリリースできます。