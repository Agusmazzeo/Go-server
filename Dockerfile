# syntax=docker/dockerfile:1.2

# Build Go application
FROM golang:1.23.5-alpine as builder

RUN apk add --update --no-cache \
    ca-certificates \
    git \
    tzdata \
    fontconfig \
    freetype \
    ttf-dejavu \
    ttf-droid \
    ttf-freefont \
    ttf-liberation \
    && rm -rf /var/cache/apk/*

WORKDIR /go/src/app
COPY go.mod go.sum ./

# Get dependencies - will also be cached if we won't change mod/sum
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/app

# ✅ Stage: Copy `wkhtmltopdf` from an external Alpine-based image
FROM surnet/alpine-wkhtmltopdf:3.16.2-0.12.6-full as wkhtmltopdf

# ✅ Final stage: Run Go application with `wkhtmltopdf`
FROM alpine:3.17

EXPOSE 5051
ENV PORT 5051

# ✅ Install fonts and required dependencies
RUN apk add --no-cache \
    libstdc++ \
    libx11 \
    libxrender \
    libxext \
    libssl1.1 \
    ca-certificates \
    fontconfig \
    freetype \
    ttf-dejavu \
    ttf-droid \
    ttf-freefont \
    ttf-liberation \
    && fc-cache -fv

# Copy `wkhtmltopdf` binaries from external image
COPY --from=wkhtmltopdf /bin/wkhtmltopdf /bin/libwkhtmltox.so /bin/

# Copy Go application and assets
COPY ./assets /assets
COPY ./settings /settings
COPY ./templates /templates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /go/bin/app /go/bin/app
COPY --from=builder /go/src/app/assets /go/src/app/assets

CMD ["/go/bin/app"]
