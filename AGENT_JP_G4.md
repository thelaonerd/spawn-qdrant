# プロジェクト - spawn-qdrant

## はじめに

本プロジェクトは、Dockerを使用してQdrantインスタンスを起動するためのシンプルなものです。Go言語で記述されており、Qdrant Dockerイメージをラッ
プするシンプルなラッパーです。このアプリケーションを使用すると、Linuxマシン上でDockerを使ってN個のqdrantインスタンスを起動することができます
。

## アプリケーションの使い方

このアプリケーションは`spawn-qdrant`と呼ばれ、`.env`ファイルから以下の引数を読み取ります。

- REST_PORT
- GRPC_PORT

`.env`ファイルが存在しない場合、アプリケーションは最初のインスタンスの起動ポートとして以下のデフォルト値を使用します。

- REST_PORT=6333
- GRPC_PORT=6334

アプリケーションの開始時、DockerまたはPodmanがインストールされているかを確認し、インストールされていればDockerを、そうでなければPodmanを使用
します。どちらもインストールされていない場合、アプリケーションはエラーメッセージを出して終了し、ユーザーにDockerまたはPodmanのインストールを
促します。

### `check`サブコマンド

`check`コマンドは、システムのRAMを検証し、起動用（インスタンスあたり256MB）および効率的な動作用（インスタンスあたり512MB）にそれぞれいくつイ
ンスタンスを実行できるかを報告します。

```bash
spawn-qdrant check
```

### `spawn`サブコマンド

起動するインスタンスの数は、`instance_count`という名前のオプション引数として`spawn`サブコマンドに渡されます。これが提供されない場合、推定ロ
ジック（`check`に類似）がデフォルトとして適用されます。

アプリケーションは`qdrant/qdrant`イメージの有無を確認し、存在しない場合はプルします。

`spawn`サブコマンドは、`qdrant_network`というDockerネットワークが存在しない場合、それを作成し、すべてのコンテナがこのネットワークに接続され
るようにします。

例えば、

```bash
spawn-qdrant spawn 2
```

を実行すると、以下のポートとストレージロケーションを持つ2つのqdrantインスタンスが起動されます。

- コンテナ1：名前はqdrant-01、ポートは6333と6334を使用し、ネットワークqdrant_network内のストレージロケーションは~/.qdrant_storage01
- コンテナ2：名前はqdrant-02、ポートは6335と6336を使用し、ネットワークqdrant_network内のストレージロケーションは~/.qdrant_storage02

このツールは、リソースの急増を緩和するために、各インスタンスの起動間に**30秒間**待機します。

ポートはqdrantインスタンスの開始ポート番号です。`instance_count`が1より大きい場合、実際に使用されるポートは、REST_PORT + 2*(instance_count
- 1) および GRPC_PORT + 2*(instance_count - 1) となります。`instance_count`が1の場合、使用されるポートはREST_PORTおよびGRPC_PORTです。

各qdrantインスタンスのストレージは、~/.qdrant_storage{instance_count}に保存されます。

### `stop`サブコマンド

`stop`サブコマンドは、`spawn`サブコマンドを使用して起動されたすべてのqdrantインスタンスを停止します。アプリケーションは以下のように使用でき
ます。

#### `stop all`サブコマンド

これは、`spawn`サブコマンドを使用して起動されたすべてのqdrantインスタンスを停止し、qdrant_networkとロックファイルを削除します。

```bash
spawn-qdrant stop all
```

#### `stop n`サブコマンド

これは、`spawn`サブコマンドを使用して起動されたn番目のqdrantインスタンスを停止します。また、それが最後のインスタンスであった場合、
qdrant_networkとロックファイルも削除します。

```bash
spawn-qdrant stop n
```

### `clean`サブコマンド

- この操作は、内部的に`stop all`サブコマンドを使用して、まず`spawn`サブコマンドで起動されたqdrantインスタンスを停止します。
- その後、アプリケーションは、ストレージロケーション`~/.qdrant_storage{instance_count}`を、日付と時刻のタイムスタンプが付いた
`~/qdrant_backup`というバックアップロケーションに単一のtar.gzファイルとしてgzip圧縮します。
- 次に、アプリケーションは、`spawn`サブコマンドで起動されたすべてのqdrantインスタンスのストレージロケーション、すなわち
`~/.qdrant_storage{instance_count}`を削除します。
- また、gzipとdeleteコマンドは、ストレージロケーションがDockerコンテナとして扱われ、所有者がrootとなるため、sudo昇格を使用して実行されます。


```bash
spawn-qdrant clean
```

### ロックファイル機構

このアプリケーションは、`~/.spawn-qdrant.lock`にロックファイルを使用し、複数の同時に実行されるスパンセッションを防ぎます。
- **ロックの作成**: `spawn`を実行する際に自動的に作成されます。
- **ロックの削除**: `stop all`、`clean`、または最後のインスタンスを`stop`した際に自動的に削除されます。