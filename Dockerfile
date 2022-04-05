FROM golang:1.17 as builder

ENV GO111MODULE=on

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/geosite2pac .

FROM scratch

WORKDIR /app

COPY --from=builder /app/geosite2pac /app/geosite2pac
COPY config /app/config

EXPOSE 8000

ENTRYPOINT ["./geosite2pac"]
CMD [ "-rule", "/app/config/rule.json", "-serve", ":8000" ]