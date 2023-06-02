# syntax=docker/dockerfile:1.2

# https://petomalina.medium.com/using-go-mod-download-to-speed-up-golang-docker-builds-707591336888

FROM golang:1.20-alpine3.17 as builder

RUN apk add --update --no-cache ca-certificates git tzdata

WORKDIR /go/src/app
COPY go.mod go.sum ./

# Get dependencies - will also be cached if we won't change mod/sum
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/app

FROM alpine:3.17

EXPOSE 5051
ENV PORT 5051

# copy the ca-certificate.crt from the build stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /go/bin/app /go/bin/app

CMD ["/go/bin/app"]
