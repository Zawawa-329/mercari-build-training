FROM golang:1.24.0-alpine3.20

# CGO を有効にする
ENV CGO_ENABLED=1

# 必要なパッケージをインストール
RUN apk add --no-cache gcc musl-dev sqlite sqlite-dev

# 作業ディレクトリを設定
WORKDIR /app

# Go モジュールをダウンロード
COPY go.mod go.sum ./
RUN go mod download

# ソースコード全体をコンテナにコピー
COPY . ./

# images ディレクトリを作成 (画像の保存場所)
RUN mkdir -p /app/images

# Go プログラムのビルド
RUN go build -o server ./cmd/api

# ユーザー・グループを作成し、必要な権限を設定
RUN addgroup -S mercari && adduser -S trainee -G mercari \
    && chown -R trainee:mercari /app/db /app/images

# ユーザーを trainee に変更
USER trainee

# サーバーを実行
CMD ["./server"]
