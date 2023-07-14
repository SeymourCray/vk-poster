FROM golang:1.20.4-alpine AS build

RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./

RUN go env -w CGO_ENABLED=0
RUN go env -w GOOS=linux
RUN go build -a -o /app/app ./cmd/main.go

FROM alpine AS run

WORKDIR /app

COPY --from=build /app/app /app
COPY --from=build /app/web /app/web
COPY --from=build /app/migrations /app/migrations
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

RUN apk add --no-cache wget
RUN wget https://yt-dl.org/downloads/latest/youtube-dl -O /usr/local/bin/youtube-dl
RUN chmod a+rx /usr/local/bin/youtube-dl
RUN apk add --no-cache ffmpeg
RUN apk add --update --no-cache python3 && ln -sf python3 /usr/bin/python
RUN python3 -m ensurepip
RUN pip3 install --no-cache --upgrade pip setuptools

ENTRYPOINT [ "/app/app" ]