FROM golang:1.18 as builder
WORKDIR /workspace
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make

FROM alpine
RUN apk add ffmpeg
WORKDIR /
COPY --from=builder /workspace/stream-manager .
ENTRYPOINT ["/stream-manager"]
