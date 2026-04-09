# spawn-qdrant - 開発者向け引き継ぎドキュメント

> **プロジェクト**: Docker/Podman を使用して複数の Qdrant インスタンスを起動・管理するための CLI ツール  
> **対象読者**: Go 開発経験 1〜2 年で、Docker/コンテナ化に習熟している開発者  
> **技術スタック**: Go 1.25+, Cobra CLI, Docker/Podman  
> **アーキテクチャ**: 関心事の分離を明確にしたレイヤードアーキテクチャ

---

## 目次

1. [新任開発者のためのクイックスタート](#1-新任開発者のためのクイックスタート)
2. [プロジェクト概要](#2-プロジェクト概要)
3. [アーキテクチャ概要](#3-アーキテクチャ概要)
4. [データフローとコンポーネント間の相互作用](#4-データフローとコンポーネント間の相互作用)
5. [エンドツーエンドのワークフロー詳細](#5-エンドツーエンドのワークフロー詳細)
6. [ディレクトリ構造](#6-ディレクトリ構造)
7. [主要なパターンと規約](#7-主要なパターンと規約)
8. [セキュリティ上の考慮事項](#8-セキュリティ上の考慮事項)
9. [テストガイド](#9-テストガイド)
10. [一般的なタスク](#10-一般的なタスク)
11. [トラブルシューティング](#11-トラブルシューティング)

---

## 1. 新任開発者のためのクイックスタート

### 1.1 クローンとビルド

```bash
# リポジトリをクローン
git clone <repository-url>
cd spawn-qdrant

# 依存関係のダウンロード
go mod download

# バイナリのビルド
go build -o spawn-qdrant main.go

# テストの実行
go test ./...
```

### 1.2 まず試すべきコマンド

```bash
# 利用可能なRAMの確認と推定可能インスタンス数の表示
./spawn-qdrant check

# 2つのインスタンスを起動 (ドライラン - 実際に行われる操作を確認)
./spawn-qdrant spawn 2

# ヘルプの表示
./spawn-qdrant --help
./spawn-qdrant spawn --help
```

### 1.3 開発ワークフロー

1. `cmd/` または `internal/` 内の関連ファイルを**変更**する。
2. **テストを実行**: `go test ./...`
3. **ビルド**: `go build -o spawn-qdrant main.go`
4. **手動テスト**: `./spawn-qdrant check`
5. **コミット**: Conventional Commits の形式に従ってコミットメッセージを記述する。

---

## 2. プロジェクト概要

**spawn-qdrant** は、Linux 上で Docker または Podman を使用して、独立した複数の Qdrant（ベクトルデータベース）インスタンスを実行することを簡素化する CLI ユーティリティです。Qdrant はベクトル類似性検索エンジンであり、本ツールを使用することで、開発、テスト、またはマルチテナント シナリオ向けに、独立したインスタンスを簡単に複数実行できます。

### 主な機能

| 機能 | 説明 |
|---------|-------------|
| **複数インスタンスの起動** | ポート番号を自動インクリメントし、N 個の独立した Qdrant インスタンスを作成 |
| **ランタイム検出** | Docker を自動的に使用し、利用不可の場合は Podman にフォールバック |
| **リソース安全性** | 起動前の RAM 推定により OOM (Out of Memory) 状態を防止 |
| **安全なクリーンアップ** | 削除前のバックアップ作成と対話形式の確認 |
| **同時実行制御** | ファイルベースのロックにより操作の競合を防止 |
| **シグナルハンドリング** | SIGINT/SIGTERM による正常なシャットダウン (Graceful Shutdown) |

### 前提条件

- **OS**: Linux (Ubuntu/Debian でテスト済み)
- **ランタイム**: Docker (推奨) または Podman がインストールされ、PATH に通っていること
- **権限**: `clean` コマンドにおいて、パスワードなしの sudo 権限（root 所有のファイルのバックアップ/削除のため）
- **Go**: バージョン 1.25 以上 (開発用)

### コンテナアーキテクチャ

起動した各インスタンスには以下が割り当てられます：
- **コンテナ名**: `qdrant-01`, `qdrant-02` など
- **REST API ポート**: 6333, 6335, 6337... (2 ずつ増加)
- **gRPC ポート**: 6334, 6336, 6338... (2 ずつ増加)
- **ストレージディレクトリ**: `~/.qdrant_storage01`, `~/.qdrant_storage02` など
- **ネットワーク**: すべてのインスタンスは `qdrant_network` (Docker bridge) に接続
- **再起動ポリシー**: `unless-stopped`

---

## 3. アーキテクチャ概要

### 3.1 ハイレベルアーキテクチャ

本プロジェクトは、以下の 3 つの明確な層からなる **レイヤードアーキテクチャ** パターンを採用しています。

```
┌─────────────────────────────────────────────────────────────────────┐
│                         レイヤー 1: プレゼンテーション (Presentation)   │
│                              cmd/ パッケージ                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │  check   │ │  spawn   │ │   stop   │ │  clean   │ │ version  │  │
│  │   cmd    │ │   cmd    │ │   cmd    │ │   cmd    │ │   cmd    │  │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘  │
└───────┼────────────┼────────────┼────────────┼────────────┼────────┘
        │            │            │            │            │
        └────────────┴────────────┴────────────┴────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       レイヤー 2: ビジネスロジック (Business Logic)       │
│                           internal/ パッケージ                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌───────────┐ │
│  │   lock/      │  │   system/    │  │  container/  │  │  config/  │ │
│  │ ファイルロック │  │  RAMチェック  │  │Docker/Podman │  │  設定読込  │ │
│  │   Create()   │  │GetAvailable()│  │   Run()      │  │   Init    │ │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └─────┬─────┘ │
└─────────┼─────────────────┼─────────────────┼────────────────┼───────┘
          │                 │                 │                │
          └─────────────────┴─────────────────┴────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      レイヤー 3: インフラストラクチャ (Infrastructure)    │
│                      外部システム / OS                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────┐ │
│  │    Docker    │  │   Podman     │  │ Linux Kernel │  │  ファイル  │ │
│  │   エンジン    │  │  (フォールバック)│  │ /proc, sysfs │  │  ~/.qdrant* │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 レイヤーの責任範囲

| レイヤー | パッケージ | 責任範囲 | 主要ファイル |
|-------|---------|----------------|-----------|
| **プレゼンテーション** | `cmd/` | CLI コマンド、フラグ解析、ユーザーインタラクション、Cobra の設定 | `root.go`, `spawn.go`, `stop.go`, `clean.go`, `check.go` |
| **ビジネスロジック** | `internal/lock/` | ファイルロックによる同時実行制御 | `lockfile.go` |
| **ビジネスロジック** | `internal/system/` | リソース確認、RAM 推定 | `resources.go` |
| **ビジネスロジック** | `internal/container/` | コンテナのライフサイクル操作 | `runtime.go` |
| **インフラストラクチャ** | Docker/Podman | コンテナランタイムの実行 | 外部バイナリ |
| **インフラストラクチャ** | Linux OS | メモリ情報、シグナル、ファイルシステム | `/proc/meminfo` |

### 3.3 コンポーネント相互作用モデル

本アプリケーションは **依存関係のない (dependency-free)** アーキテクチャを採用しています。ビジネスロジック パッケージ同士は直接インポートせず、関数のパラメータを通じて依存関係を受け取ります。

```go
// cmd/spawn.go - オーケストレーターパターン
func spawnWorkflow() error {
    // 1. ロックの取得
    if err := lock.Create(); err != nil {
        return err
    }
    defer lock.Remove() // クリーンアップパターン

    // 2. リソースチェック
    ramMB, _ := system.GetAvailableRAM()
    
    // 3. コンテナ操作
    container.EnsureImage("qdrant/qdrant")
    container.CreateNetwork("qdrant_network")
    
    // 4. インスタンス作成ループ
    for i := 0; i < count; i++ {
        container.RunQdrant(config)
    }
}
```

**基本原則**: `cmd/` 内のコマンドが `internal/` パッケージの呼び出しをオーケストレートします。`internal/` 内の各パッケージは独立しており、単一の責任に集中しています。

---

## 4. データフローとコンポーネント間の相互作用

### 4.1 `spawn` コマンドのデータフロー

ユーザーが `spawn-qdrant spawn 2` を実行した際のデータフローは以下の通りです。

```
┌────────┐                                                    
│  ユーザー  │ CLI 入力: "spawn 2"                               
└───┬────┘                                                    
    │                                                         
    ▼                                                         
┌──────────────────────────────────────────────────────────┐
│  レイヤー 1: プレゼンテーション (cmd/)                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  root.go                                            │  │
│  │  - フラグ解析: --rest-port, --grpc-port           │  │
│  │  - viper 設定へのバインド                             │  │
│  │  - spawn サブコマンドへルーティング                        │  │
│  └─────────────────────────────────────────────────────┘  │
│                          │                                │
│                          ▼                                │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  spawn.go                                           │  │
│  │  - instance_count のバリデーション                          │  │
│  │  - シグナルハンドリングの設定 (SIGINT/SIGTERM)             │  │
│  │  - internal パッケージの呼び出し                           │  │
│  └─────────────────────────────────────────────────────┘  │
└──────────────────────────┬───────────────────────────────┘
                           │ 関数呼び出し
                           ▼
┌──────────────────────────────────────────────────────────┐
│  レイヤー 2: ビジネスロジック (internal/)                    │
│                                                          │
│  ┌─────────────┐    ┌─────────────┐    ┌──────────────┐ │
│  │  lock/      │    │  system/    │    │  container/  │ │
│  │  Create()   │    │GetAvailable │    │  EnsureImage │ │
│  │  ├─▶ ファイル   │    │  RAM()      │    │  ├─▶ docker   │ │
│  │  │  ~/.spawn │    │  ├─▶ /proc  │    │  │   inspect  │ │
│  │  │  -qdrant  │    │  │  /meminfo │    │  │            │ │
│  │  │  .lock    │    │  │           │    │  └─▶ docker   │ │
│  │  │           │    │  └─▶ 返却    │    │     pull      │ │
│  │  └─▶ bool    │    │     uint64   │    │               │ │
│  │     (成功)   │    │              │    │  CreateNetwork│ │
│  │              │    │  Estimate()  │    │  ├─▶ docker   │ │
│  │  Remove()    │    │  ├─▶ 計算    │    │  │   network   │ │
│  │  ├─▶ os.     │    │  │  起動/    │    │  │   create    │ │
│  │  │  Remove() │    │  │  効率的   │    │  │             │ │
│  │  │           │    │  │           │    │  └─▶ 返却     │ │
│  │  └─▶ error   │    │  └─▶ 返却    │    │     string    │ │
│  │     (nil ok) │    │     (最大 S, E)│    │               │ │
│  └─────────────┘    └─────────────┘    │  RunQdrant()  │ │
│                                          │  ├─▶ docker   │ │
│                                          │  │   run       │ │
│                                          │  │   [config]  │ │
│                                          │  │             │ │
│                                          │  └─▶ 返却     │ │
│                                          │     error     │ │
│                                          └──────────────┘ │
└──────────────────────────┬───────────────────────────────┘
                           │ システムコール / exec.Command
                           ▼
┌──────────────────────────────────────────────────────────┐
│  レイヤー 3: インフラストラクチャ                                │
│                                                          │
│  ┌──────────┐  ┌────────────┐  ┌─────────────────────┐  │
│  │  Docker  │  │   Podman   │  │    Linux OS         │  │
│  │  バイナリ  │  │  (フォールバック)│  │                     │  │
│  │          │  │            │  │  ┌───────────────┐  │  │
│  │ コマンド:│  │  コマンド: │  │  │/proc/meminfo  │  │  │
│  │  - run   │  │   - run    │  │  │  (RAM データ)   │  │  │
│  │  - stop  │  │   - stop   │  │  └───────────────┘  │  │
│  │  - rm    │  │   - rm     │  │                     │  │
│  │  - pull  │  │   - pull   │  │  ┌───────────────┐  │  │
│  │  - ps    │  │   - ps     │  │  │ シグナル:       │  │  │
│  │  - network│  │   - network│  │  │ SIGINT/TERM   │  │  │
│  │    create│  │     create │  │  │               │  │  │
│  │          │  │            │  │  │ コンテキスト:    │  │  │
│  │ 出力:    │  │  出力:     │  │  │ ctx.Done()    │  │  │
│  │ コンテナ  │  │  コンテナ  │  │  └───────────────┘  │  │
│  │ ネットワーク │  │  ネットワーク  │  │                     │  │
│  └──────────┘  └────────────┘  └─────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

### 4.2 状態管理

アプリケーションは最小限の状態を維持します。

| 状態タイプ | 保存場所 | 目的 |
|------------|----------|---------|
| **ロックファイル** | `~/.spawn-qdrant.lock` | `spawn` 操作の同時実行を防止 |
| **設定ファイル** | `~/.spawn-qdrant.yaml` | ユーザー設定 (使用頻度は低い) |
| **ストレージディレクトリ** | `~/.qdrant_storageNN` | コンテナのデータボリューム |
| **バックアップディレクトリ** | `~/qdrant_backup/` | `clean` コマンドによるストレージのアーカイブ |
| **Docker 状態** | Docker デーモン | コンテナおよびネットワークのライフサイクル管理 |

**状態の遷移**:
1. **ロック取得** $\rightarrow$ 操作開始
2. **コンテナ作成** $\rightarrow$ Docker デーモンが状態を管理
3. **ロック解放** $\rightarrow$ 成功またはエラー時 (defer による)
4. **クリーンアップ** $\rightarrow$ `stop` または `clean` コマンドによりコンテナとロックを削除

---

## 5. エンドツーエンドのワークフロー詳細

### 5.1 ワークフロー 1: インスタンスの起動 (`spawn-qdrant spawn 3`)

**ビジネス目的**: 独立したストレージとインクリメントされるポートを持つ N 個の Qdrant コンテナインスタンスを作成する。

**エントリポイント**: `cmd/spawn.go`

**ステップバイステップの実行フロー**:

#### フェーズ 1: 初期化

**33-44行目: シグナルハンドリングの設定**
```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()
```
> SIGINT/SIGTERM をリッスンして正常にシャットダウンするためのコンテキストを作成します。起動中にユーザーが Ctrl+C を押すと、コンテキストのキャンセルがトリガーされ、クリーンアップが実行されます。

**46-58行目: 引数解析**
```go
n := 1 // デフォルト
if len(args) > 0 {
    parsed, err := strconv.Atoi(args[0])
    if err != nil || parsed < 1 {
        return fmt.Errorf("instance_count must be a positive integer")
    }
    n = parsed
}
```
> 入力値をバリデーションします（正の整数であること）。無効な入力に対しては分かりやすいエラーメッセージを返します。

**61-70行目: ロックの取得**
```go
if err := lock.Create(); err != nil {
    return fmt.Errorf("failed to acquire lock: %w", err)
}
defer func() {
    if cleanupLock {
        lock.Remove()
    }
}()
```
> **重要**: `spawn` 操作の同時実行を防止します。`~/.spawn-qdrant.lock` ファイルを `O_EXCL` フラグ付きで作成します（既に存在する場合は失敗します）。パニック時でも必ずロックが削除されるよう、クリーンアップを `defer` しています。

#### フェーズ 2: リソースの検証

**72-85行目: RAM チェック**
```go
ramMB, err := system.GetAvailableRAM()
if err != nil {
    return fmt.Errorf("failed to get available RAM: %w", err)
}

maxStartup, maxEfficient := system.EstimateInstances(ramMB)
if uint64(n) > maxEfficient {
    return fmt.Errorf("insufficient RAM for %d instances (max efficient: %d, max startup: %d)", 
        n, maxEfficient, maxStartup)
}
```
> `/proc/meminfo` の `MemAvailable` を読み取り、キャパシティを計算します（起動時 256MB/台、効率的運用 512MB/台）。これにより OOM 状態を防止します。

#### フェーズ 3: コンテナの準備

**95-101行目: イメージチェック**
```go
if err := container.EnsureImage("qdrant/qdrant"); err != nil {
    lock.Remove() // リターン前の手動クリーンアップ
    return fmt.Errorf("failed to ensure qdrant image: %w", err)
}
```
> イメージがローカルに存在するか確認し、存在しない場合は `docker pull qdrant/qdrant` を実行します。エラー時は (defer が実行される前に) 手動でロックを削除します。

**103行目: ネットワーク作成**
```go
_ = container.CreateNetwork("qdrant_network")
```
> べき等なネットワーク作成です。複数回呼び出しても安全です（Docker 側で重複が処理されます）。

#### フェーズ 4: インスタンス作成ループ

**115-170行目: メインループ**
```go
for i := 0; i < n; i++ {
    // ポート計算
    restPort := startRest + (2 * i)  // 6333, 6335, 6337...
    grpcPort := startGrpc + (2 * i)  // 6334, 6336, 6338...
    suffix := fmt.Sprintf("%02d", i+1)  // "01", "02", "03"...
    
    // コンテキストのキャンセルチェック
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // コンテナ作成
    containerName := fmt.Sprintf("qdrant-%s", suffix)
    storageDir := filepath.Join(homeDir, fmt.Sprintf(".qdrant_storage%s", suffix))
    
    err := container.RunQdrant(container.QdrantConfig{
        Name:       containerName,
        Network:    "qdrant_network",
        RestPort:   restPort,
        GrpcPort:   grpcPort,
        StorageDir: storageDir,
    })
    
    // インスタンス間の待機時間 (最後以外)
    if i < n-1 {
        select {
        case <-time.After(30 * time.Second):
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```
> **主要な動作**:
- ポートは 2 ずつインクリメントされ、ポート競合を回避します。
- 名前はゼロ埋めされます (`qdrant-01` となり `qdrant-1` にはなりません)。
- インスタンス間に 30 秒の待機時間を設け、Qdrant の初期化を待ちます。
- 各チェックポイントでコンテキストのキャンセルを確認します。

#### コンテナランタイム層

**場所**: `internal/container/runtime.go`

**ランタイム検出** (Docker または Podman を自動検出):
```go
func InitRuntime() error {
    if isCommandAvailable("docker") {
        containerRuntime = Docker
        return nil
    }
    if isCommandAvailable("podman") {
        containerRuntime = Podman
        return nil
    }
    return fmt.Errorf("neither docker nor podman is installed")
}
```

**コンテナ作成**:
```go
func RunQdrant(cfg QdrantConfig) error {
    // セキュリティ: パスインジェクションを防止
    if strings.Contains(cfg.StorageDir, ":") {
        return fmt.Errorf("invalid storage directory: path cannot contain ':'")
    }

    return runCommand("run", "-d",
        "--name", cfg.Name,
        "--net", cfg.Network,
        "--restart", "unless-stopped",
        "-p", fmt.Sprintf("%d:6333", cfg.RestPort),
        "-p", fmt.Sprintf("%d:6334", cfg.GrpcPort),
        "-v", fmt.Sprintf("%s:/qdrant/storage", cfg.StorageDir),
        "qdrant/qdrant",
    )
}
```

**コマンド実行パターン**:
```go
func runCommand(args ...string) error {
    cmd := exec.Command(string(containerRuntime), args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
```

### 5.2 ワークフロー 2: インスタンスの停止 (`spawn-qdrant stop all` または `spawn-qdrant stop 2`)

**ビジネス目的**: Qdrant コンテナとネットワークを正常に停止し、削除する。

**エントリポイント**: `cmd/stop.go`

**Stop All フロー** (`stopAll` 関数):
```go
func stopAll(cmd *cobra.Command) error {
    // 1. "qdrant-" プレフィックスを持つコンテナをリストアップ
    targets, err := container.ListContainerNames("qdrant-")
    
    // 2. 各コンテナを停止し削除
    for _, name := range targets {
        container.StopAndRemoveContainer(name)
    }
    
    // 3. ネットワークを削除
    _ = container.RemoveNetwork("qdrant_network")
    
    // 4. ロックファイルを削除
    return lock.Remove()
}
```

**単一インスタンスの停止** (`stopInstance` 関数):
```go
func stopInstance(cmd *cobra.Command, n int) error {
    name := fmt.Sprintf("qdrant-%02d", n)
    stopAndRemove(cmd, name)
    
    // インスタンスが残っているか確認
    anyRemaining, _ := container.HasRunningContainers("name=qdrant-")
    if !anyRemaining {
        _ = container.RemoveNetwork("qdrant_network")
        _ = lock.Remove()  // 最後のインスタンスであればロックを削除
    }
}
```

### 5.3 ワークフロー 3: クリーンアップとバックアップ (`spawn-qdrant clean`)

**ビジネス目的**: バックアップを伴う破壊的なクリーンアップ。コンテナを停止し、データを tar.gz にバックアップしてからストレージを削除する。

**エントリポイント**: `cmd/clean.go`

**実行フロー**:
```go
func cleanWorkflow(cmd *cobra.Command) error {
    // 1. 対話的な確認 (TTY かつ --force がない場合)
    if !viper.GetBool("force") && isatty(os.Stdin) {
        // ユーザーに確認: "本当によろしいですか？"
    }
    
    // 2. 全インスタンスを停止
    stopAll(cmd)
    
    // 3. バックアップファイルの作成
    backupFile := filepath.Join(homeDir, "qdrant_backup", 
                 fmt.Sprintf("backup_%s.tar.gz", timestamp))
    
    // 4. ストレージディレクトリのバリデーションとフィルタリング
    validatedMatches := filterStorageDirs(cmd, matches)
    
    // 5. sudo を使用してバックアップ (root所有のファイルであるため)
    tarCmd := exec.CommandContext(ctx, "sudo", "tar", "-czf", backupFile, "--", dirs...)
    
    // 6. sudo を使用して削除
    rmCmd := exec.CommandContext(rmCtx, "sudo", "rm", "-rf", "--", dirs...)
}
```

**セキュリティ: シンボリックリンクのバリデーション**:
```go
func filterStorageDirs(cmd *cobra.Command, matches []string) []string {
    var validated []string
    for _, match := range matches {
        info, err := os.Lstat(match)
        
        // シンボリックリンクを拒否 (権限昇格攻撃を防止)
        if info.Mode()&os.ModeSymlink != 0 {
            logInfo(cmd, "Warning: %s is a symbolic link, skipping", match)
            continue
        }
        
        // ディレクトリである必要がある
        if !info.IsDir() {
            continue
        }
        validated = append(validated, match)
    }
    return validated
}
```

---

## 6. ディレクトリ構造

```
spawn-qdrant/
├── main.go                          # エントリポイント、終了コードのハンドリング
├── go.mod                           # Go モジュール定義
├── go.sum                           # 依存関係のチェックサム
├── README.md                        # ユーザー向けドキュメント
├── HANDOVER.md                      # この引き継ぎドキュメント
├── architecture-diagram.drawio      # アーキテクチャ可視化図
│
├── cmd/                             # Cobra CLI コマンド (プレゼンテーション層)
│   ├── root.go                      # ルートコマンド、設定初期化、ロギングヘルパー
│   │                                # - initConfig(): viper の設定
│   │                                # - logInfo(), logWarn(): 一貫したロギング
│   ├── spawn.go                     # インスタンス起動コマンド (複雑なワークフロー)
│   ├── stop.go                      # インスタンス停止コマンド
│   ├── clean.go                     # クリーンアップ/バックアップコマンド (最複雑)
│   ├── check.go                     # RAM チェックコマンド
│   ├── version.go                   # バージョン表示コマンド
│   ├── completion.go                # シェル補完生成
│   └── clean_test.go                # clean コマンドのテスト
│
└── internal/                        # 内部アプリケーションコード (ビジネスロジック層)
    ├── container/
    │   └── runtime.go               # Docker/Podman 抽象化
    │                                  # - InitRuntime(): 自動検出
    │                                  # - RunQdrant(): コンテナ作成
    │                                  # - StopAndRemoveContainer(): クリーンアップ
    │                                  # - EnsureImage(): 必要に応じてプル
    │                                  # - CreateNetwork(), RemoveNetwork()
    ├── lock/
    │   └── lockfile.go              # ファイルベースのロック
    │                                  # - Create(): アトミックなロック取得
    │                                  # - Remove(): 安全なロック解放
    │                                  # - Exists(): 状態確認
    ├── system/
    │   ├── resources.go             # RAM 確認と推定
    │                                  # - GetAvailableRAM(): /proc/meminfo の解析
    │                                  # - EstimateInstances(): キャパシティ計算
    │   └── resources_test.go        # 計算ロジックのユニットテスト
    └── config/
        └── config.go                # 設定の読み込み (viper 連携)
                                     # - 現在は最小限の使用
```

---

## 7. 主要なパターンと規約

### 7.1 コマンドパターン (Cobra)

各コマンドは以下の構造に従います。

```go
var commandNameCmd = &cobra.Command{
    Use:   "command-name [args]",
    Short: "簡潔な説明",
    Long:  `詳細な複数行の説明`,
    Args:  cobra.ExactArgs(1),  // または MaximumNArgs, RangeArgs など
    RunE: func(cmd *cobra.Command, args []string) error {
        // エラーハンドリングの一貫性を保つため、エラーを返却する
        return nil
    },
}

func init() {
    rootCmd.AddCommand(commandNameCmd)
    // ここにフラグを追加
    commandNameCmd.Flags().BoolVarP(&force, "force", "f", false, "説明")
}
```

### 7.2 エラーハンドリングパターン

**常にコンテキストを付けてエラーをラップする**:
```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

**エラー時のクリーンアップ** (極めて重要なパターン):
```go
if err := riskyOperation(); err != nil {
    lock.Remove()  // 失敗時は必ずリソースをクリーンアップする
    return err
}
```

**Deferred クリーンアップ** (正常系パス用):
```go
lock.Create()
defer lock.Remove()  // パニックが発生しても実行される
```

### 7.3 コンテキストキャンセルパターン

**セットアップ**:
```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()
```

**チェックポイントパターン** (キャンセルを尊重する):
```go
for i := 0; i < n; i++ {
    select {
    case <-ctx.Done():
        // ユーザーによる中断 - クリーンアップして終了
        if i == 0 {
            lock.Remove()
        }
        return ctx.Err()
    default:
        // 操作を継続
    }
    
    // キャンセルをサポートした長時間操作
    select {
    case <-time.After(30 * time.Second):
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

### 7.4 コンテナランタイムの抽象化

プロジェクトでは Docker/Podman を抽象化し、簡単に切り替えられるようにしています。

```go
type Runtime string

const (
    Docker Runtime = "docker"
    Podman Runtime = "podman"
)

var containerRuntime Runtime

// すべての操作は containerRuntime 変数を使用する
func runCommand(args ...string) error {
    cmd := exec.Command(string(containerRuntime), args...)
    return cmd.Run()
}
```

**理由**: 他のコンテナランタイム (containerd, cri-o など) への拡張を容易にするためです。

### 7.5 設定の階層構造

設定の優先順位（高い順）：
1. コマンドラインフラグ (`--rest-port`)
2. 環境変数 (`SPAWN_QDRANT_REST_PORT`)
3. 設定ファイル (`~/.spawn-qdrant.yaml`)
4. デフォルト値 (ハードコード)

```go
// root.go の init() 内
viper.SetEnvPrefix("SPAWN_QDRANT")
viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
viper.AutomaticEnv()

// コマンド内での使用
value := viper.GetString("rest-port")  // どのソースからも読み取り可能
```

### 7.6 命名規約

| 種類 | パターン | 例 |
|------|---------|---------|
| コマンド | `[action]Cmd` | `spawnCmd`, `stopCmd` |
| 結果/DTO | PascalCase | `QdrantConfig` |
| プライベート関数 | camelCase | `stopAll`, `stopInstance` |
| パッケージ名 | 小文字、アンダースコアなし | `container`, `lock` |
| 定数 | PascalCase | `Docker`, `Podman` |
| 変数 | camelCase | `containerRuntime`, `ramMB` |

---

## 8. セキュリティ上の考慮事項

### 8.1 引数インジェクションの防止

ユーザー提供の引数の前には **必ず `--` セパレーター** を使用してください。

```go
// BAD - インジェクションに脆弱
exec.Command("tar", "-czf", backupFile, userPath)

// GOOD - セパレーターを使用
exec.Command("tar", "-czf", backupFile, "--", userPath)
```

**コードベースでの例**:
```go
// runtime.go
docker pull -- qdrant/qdrant

// clean.go
sudo tar -czf backup.tar.gz -- ~/.qdrant_storage01
sudo rm -rf -- ~/.qdrant_storage01
```

### 8.2 パストラバーサルの防止

**RunQdrant 内のバリデーション**:
```go
if strings.Contains(cfg.StorageDir, ":") {
    return fmt.Errorf("invalid storage directory: path cannot contain ':'")
}
```

**理由**: コロンはパスセパレーターとして使用されたり、一部のインジェクション手法に利用されるためです。

### 8.3 シンボリックリンク攻撃の防止

```go
// filterStorageDirs() 内
if info.Mode()&os.ModeSymlink != 0 {
    logInfo(cmd, "Warning: %s is a symbolic link, skipping", match)
    continue
}
```

**攻撃シナリオ**: 攻撃者が `~/.qdrant_storage01 -> /etc/critical-files` のようなシンボリックリンクを作成した場合、`clean` コマンドが誤ってシステムファイルを削除してしまう可能性があります。

### 8.4 タイムアウト保護

```go
// CI や非対話環境でのハングアップを防止
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
defer cancel()

tarCmd := exec.CommandContext(ctx, "sudo", tarArgs...)
```

### 8.5 ファイル権限

```go
// ロックファイル: 所有者のみ読み書き可能 (0600 = rw-------)
f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)

// ストレージディレクトリ: 所有者とグループに読み取り/実行権限 (0755 = rwxr-xr-x)
os.MkdirAll(storageDir, 0755)
```

### 8.6 権限昇格への意識

`clean` コマンドで `sudo` が必要な理由：
1. Docker コンテナはデフォルトで root として動作する。
2. コンテナは `~/.qdrant_storageNN` 内に root 所有のファイルを作成する。
3. 通常のユーザーは root 所有のファイルを削除できない。
4. 解決策: `sudo rm -rf -- dirs...`

**安全性**: sudo 操作の前には必ずパスのバリデーション（シンボリックリンクチェック、パストラバーサルチェック）を行ってください。

---

## 9. テストガイド

### 9.1 テストの実行

```bash
# すべてのテストを実行
go test ./...

# 詳細出力を表示
go test -v ./...

# 特定のパッケージのテストを実行
go test ./internal/system/...

# カバレッジを表示
go test -cover ./...

# レース検出を有効化
go test -race ./...
```

### 9.2 テストパターン

**テーブル駆動テスト** (`resources_test.go` より):
```go
func TestEstimateInstances(t *testing.T) {
    tests := []struct {
        ramMB         uint64
        wantStartup   uint64
        wantEfficient uint64
    }{
        {0, 0, 0},
        {256, 1, 0},
        {512, 2, 1},
        {1024, 4, 2},
        {4096, 16, 8},
    }
    for _, tt := range tests {
        gotStartup, gotEfficient := EstimateInstances(tt.ramMB)
        if gotStartup != tt.wantStartup || gotEfficient != tt.wantEfficient {
            t.Errorf("EstimateInstances(%d) = (%d, %d), want (%d, %d)", 
                tt.ramMB, gotStartup, gotEfficient, tt.wantStartup, tt.wantEfficient)
        }
    }
}
```

**ファイルシステムテスト** (`clean_test.go` より):
```go
func TestFilterStorageDirs(t *testing.T) {
    // 一時ディレクトリを作成
    tmpDir, _ := os.MkdirTemp("", "spawn-qdrant-test")
    defer os.RemoveAll(tmpDir)  // テスト後にクリーンアップ
    
    // セットアップ: テストファイルとシンボリックリンクを作成
    regularDir := filepath.Join(tmpDir, "regular")
    os.Mkdir(regularDir, 0755)
    
    symlinkDir := filepath.Join(tmpDir, "symlink")
    os.Symlink("/etc", symlinkDir)
    
    // 実行: 関数を呼び出す
    matches := []string{regularDir, symlinkDir}
    validated := filterStorageDirs(nil, matches)
    
    // アサート: 結果を確認
    if len(validated) != 1 || validated[0] != regularDir {
        t.Errorf("Expected only regular dir, got %v", validated)
    }
}
```

### 9.3 インテグレーションテスト戦略

**課題**: コンテナ操作のテストには Docker/Podman が必要です。

**アプローチ 1: モック化** (ユニットテスト用)
```go
type ContainerRunner interface {
    RunQdrant(cfg QdrantConfig) error
    ListContainers() ([]string, error)
}

type MockRunner struct {
    Containers []string
}

func (m *MockRunner) RunQdrant(cfg QdrantConfig) error {
    m.Containers = append(m.Containers, cfg.Name)
    return nil
}
```

**アプローチ 2: ビルドタグ** (インテグレーションテスト用)
```go
// +build integration

func TestIntegrationSpawn(t *testing.T) {
    if !container.IsRuntimeAvailable() {
        t.Skip("Docker/Podman not available")
    }
    // 実際のコンテナ操作を実行
}
```

実行方法: `go test -tags=integration ./...`

### 9.4 コミット前のテストチェックリスト

- [ ] `go test ./...` がパスすること
- [ ] `go build` が警告なしに成功すること
- [ ] `go vet ./...` がクリーンであること
- [ ] `gofmt -d .` でフォーマットの問題がないこと
- [ ] 新しい関数に対応するテストカバレッジがあること
- [ ] エッジケース (0 インスタンス, 最大 RAM など) が処理されていること

---

## 10. 一般的なタスク

### 10.1 新しいコマンドの追加

**ステップ 1**: `cmd/newcommand.go` を作成

```go
package cmd

import "github.com/spf13/cobra"

var newCmd = &cobra.Command{
    Use:   "newcommand [arg]",
    Short: "簡潔な説明",
    Long:  `詳細な説明`,
    Args:  cobra.ExactArgs(1),  // または他のバリデーション
    RunE: func(cmd *cobra.Command, args []string) error {
        // 実装
        return nil
    },
}

func init() {
    rootCmd.AddCommand(newCmd)
    
    // フラグを追加
    newCmd.Flags().String("option", "default", "説明")
    viper.BindPFlag("option", newCmd.Flags().Lookup("option"))
}
```

**ステップ 2**: 手動でテスト
```bash
go build -o spawn-qdrant main.go
./spawn-qdrant newcommand test
```

**ステップ 3**: `cmd/newcommand_test.go` にテストを追加

### 10.2 新しいコンテナ操作の追加

**場所**: `internal/container/runtime.go`

```go
func NewOperation(name string) error {
    return runCommand("operation", "--", name)
}
```

**パターン**: 常に `--` セパレーターを使用し、ラップしたエラーを返してください。

### 10.3 設定オプションの追加

**場所**: `cmd/root.go` の `init()` 内

```go
rootCmd.PersistentFlags().String("new-option", "default", "説明")
viper.BindPFlag("new-option", rootCmd.PersistentFlags().Lookup("new-option"))
```

**使用方法**:
```go
// 任意のコマンド内で
value := viper.GetString("new-option")
```

### 10.4 リリースバイナリのビルド

```bash
# バージョン情報を込めたプロダクションビルド
go build -ldflags "-X github.com/thelaonerd/spawn-qdrant/cmd.Version=1.0.0 \
  -X github.com/thelaonerd/spawn-qdrant/cmd.Commit=$(git rev-parse --short HEAD) \
  -X github.com/thelaonerd/spawn-qdrant/cmd.BuildDate=$(date -u +%Y-%m-%d)" \
  -o spawn-qdrant main.go

# 確認
./spawn-qdrant version
```

---

## 11. トラブルシューティング

### 11.1 よくある問題

| 問題 | 原因 | 解決策 |
|-------|-------|----------|
| **"Lock file exists"** | 前回の実行がクラッシュした | `rm ~/.spawn-qdrant.lock` |
| **"neither docker nor podman"** | ランタイムが未インストールまたは PATH にない | Docker/Podman をインストールし `docker ps` が動作することを確認 |
| **"Permission denied during clean"** | ストレージが root 所有である (コンテナ由来) | sudo で実行するか、パスワードなし sudo を設定 |
| **"Port already in use"** | 他のサービスや以前のコンテナと競合している | `docker ps` で確認し、`docker rm` で競合を削除 |
| **"Insufficient RAM"** | 要求したインスタンス数が利用可能メモリを超えている | まず `check` コマンドを実行して制限を確認 |
| **"docker: command not found"** | 非ログインシェルで Docker が PATH にない | フルパスを使用するか、`~/.bashrc` 等で PATH を設定 |

### 11.2 終了コード

`main.go` で定義されています。

| コード | 意味 | 返却されるケース |
|------|---------|---------------|
| 0 | 成功 | コマンドが正常に完了した |
| 1 | 一般的な失敗 | 実行中に予期せぬエラーが発生した |
| 64 | 使用法エラー | 引数が無効、または必須フラグが不足している |
| 65 | データエラー | RAM 不足、ポート競合 |
| 71 | システムエラー | sudo 失敗、ツール不足、ロック問題 |
| 130 | キャンセル | ユーザーが Ctrl+C/SIGINT で中断した |

**スクリプトでの利用例**:
```bash
./spawn-qdrant spawn 5
if [ $? -eq 65 ]; then
    echo "RAM が不足しています。インスタンス数を減らして試してください"
fi
```

### 11.3 デバッグモード

```bash
# 詳細出力 (コマンドに実装されている場合)
./spawn-qdrant -v spawn 2

# または Go のデバッグ出力を利用
DEBUG=1 ./spawn-qdrant spawn 2

# ログの確認
journalctl -u docker  # systemd を使用している場合

# コンテナログの確認
docker logs qdrant-01
```

### 11.4 回復手順

**シナリオ 1: ロックファイルが残ったままになった**
```bash
# 実際にプロセスが動作しているか確認
ps aux | grep spawn-qdrant

# 動作していない場合は削除して安全
rm ~/.spawn-qdrant.lock
```

**シナリオ 2: 部分的な起動 (一部のコンテナのみ作成された)**
```bash
# 存在ものを確認
docker ps -a | grep qdrant

# 停止してクリーンアップ
./spawn-qdrant stop all
# または手動で:
docker rm -f qdrant-01 qdrant-02
```

**シナリオ 3: ネットワークの競合**
```bash
# 既存のネットワークを確認
docker network ls | grep qdrant

# スタックしている場合は削除
docker network rm qdrant_network
```

---

## まとめ

### 新任開発者への重要ポイント

1. **アーキテクチャは階層化されている**: `cmd/` (プレゼンテーション) $\rightarrow$ `internal/` (ビジネスロジック) $\rightarrow$ Docker/ホスト (インフラストラクチャ)
2. **外部コマンドでユーザーパスを使用する際は必ず `--` を付ける**: (インジェクション攻撃対策)
3. **エラー時のリソースクリーンアップ**: ロックには `defer` を使用し、早期リターン前には手動クリーンアップを行う。
4. **コンテキストキャンセルを尊重する**: ループ内やブロッキング操作で `ctx.Done()` をチェックする。
5. **Cobra パターンに従う**: 各コマンドは `init()` で登録される `*cobra.Command` である。
6. **コミット前にテストする**: `go test ./...`, `go vet`, `gofmt` を実行する。

### アーキテクチャの強み

- **クリーンな分離**: CLI とビジネスロジックが切り離されている。
- **ランタイムに依存しない**: Docker/Podman の抽象化。
- **セキュリティ意識**: シンボリックリンクチェック、引数セパレーター、パスバリデーション。
- **シグナル対応**: 中断時の正常なシャットダウン。
- **十分なテスト**: RAM 計算やパスフィルタリングなどの重要パスにユニットテストがある。

### 今後の拡張候補

| 機能 | 複雑度 | 備考 |
|---------|------------|-------|
| `logs` コマンドの追加 | 低 | `docker logs` のラッパー |
| `status` コマンドの追加 | 低 | 動作中のインスタンス一覧とヘルスチェックを表示 |
| カスタム Qdrant バージョン指定 | 中 | `--version v1.2.3` フラグの導入 |
| 起動後のヘルスチェック | 中 | REST ポートへの HTTP チェック |
| Podman 特有のネットワーク対応 | 中 | ルートレス Podman のネットワーキングは異なるため |
| Docker Compose エクスポート | 高 | `docker-compose.yml` を生成 |
| リモートデプロイ | 高 | リモートの Docker デーモンへ SSH 接続 |

---

**ドキュメントバージョン**: 1.0  
**最終更新日**: 2026-04-09  
**作成者**: 開発者引き継ぎチーム