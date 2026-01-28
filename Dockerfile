# ベースとなるイメージ（最新の安定版Go環境）
FROM golang:1.25

# コンテナ内の作業ディレクトリを設定
WORKDIR /app

# ホストのファイルをコンテナ内にコピーするための準備
# (docker-composeでボリュームマウントするため、実質的には開発中に使いませんが、ビルド用に記述します)
COPY . .

# 開発に便利なツールを入れておく（今回はgitなどが入っていれば十分）
RUN apt-get update && apt-get install -y git

# コンテナが起動し続けるようにシェルを動かしておく
CMD ["bash"]