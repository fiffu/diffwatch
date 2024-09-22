# syntax=docker/dockerfile:1

FROM golang:1.22

WORKDIR /app

COPY . ./
RUN go build -o ./diffwatch

ENTRYPOINT ["./diffwatch"]
