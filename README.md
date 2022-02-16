# tpf2-tpnetmap-toolkit

TransportFever2 のワールドデータから svg のマップ画像を作成するツールキットです。

![サンプル画像](https://imgur.com/QniphV1.png)

## 1. 導入方法

(1) [Release ページ](https://github.com/nosrith/tpf2-tpnetmap-toolkit/releases) から zip ファイルをダウンロードします。

(2) ダウンロードした zip ファイルを適当な場所に展開します。

(3) `nosrith_tpnetmap_export_1` フォルダを TpF2 の MOD フォルダに移動します。

## 2. 使い方

このツールキットは次の3つのツールで構成されています。

**hillshade**: ハイトマップから陰影起伏図を作成します。マップ画像の背景として使用します。背景画像を別に用意している場合、または背景画像を使わない場合は不要です。

**TpNetMap Export** (nosrith_tpnetmap_export_1): ゲーム中のマップデータをファイルに書き出します。TpF2のMODとして動作します。

**tpnetmap**: ファイルに書き出されたマップデータをもとにマップ画像を作成します。

### 2.1. hillshade

(1) ハイトマップの画像ファイルを用意し、`data` フォルダの下に `heightmap.png` として配置します。画像ファイルのパスを設定ファイルの `imagePath` で指定することもできます。

(2) 設定ファイル `hillshade_settings.yaml` を編集します。`minHeight`、 `maxHeight`（ゲーム内の標高の最小値と最大値）はマップごとに設定が必要です。

(3) `hillshade.exe` を実行します。`data` フォルダの下に陰影起伏図の画像ファイル `hillshade.png` が出力されます。出力先のパスを設定ファイルの `outPath` で指定することもできます。

### 2.2. TpNetMap Export

(1) マップのロード時に `TpNetMap Export` の MOD を有効にします。

(2) ロード後、下メニューに表示される `TpNetMap Export` ボタンをクリックします。`TransportFever2.exe` のあるフォルダにマップデータファイル `map.yaml` が出力されます。

### 2.3. tpnetmap

(1) マップデータファイル `map.yaml` を `data` フォルダの下に配置します。ファイルのパスを設定ファイルの `mapPath` で指定することもできます。

(2) 設定ファイル `tpnetmap_settings.yaml` を編集します。背景画像のパスは `backgroundPath` で指定できます。

(3) `tpnetmap.exe` を実行します。`data` フォルダの下にマップ画像ファイル `map.svg` が出力されます。出力先のパスを設定ファイルの `outPath` で指定することもできます。
