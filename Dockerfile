# First Stage: Build the application
FROM golang:1.23.4-alpine3.21 AS builder

LABEL maintainer="zzy1998"

WORKDIR /app

COPY go.mod ./

RUN go mod download

COPY . .

RUN go build -o document-converter .

# Second Stage: Copy the binary and required files to a new image
FROM ubuntu:24.04 AS runner

ENV GDK_DPI_SCALE=1
ENV GDK_SCALE=1

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends libreoffice fonts-thai-tlwg && rm -rf /var/lib/apt/lists/*

# Built-in fonts (baked into the image)
COPY fonts /usr/share/fonts/custom

# fontconfig: also scan /app/fonts so mounted fonts are picked up at runtime
RUN mkdir -p /app/fonts && \
    printf '<?xml version="1.0"?>\n<!DOCTYPE fontconfig SYSTEM "fonts.dtd">\n<fontconfig>\n  <dir>/app/fonts</dir>\n</fontconfig>\n' \
    > /etc/fonts/conf.d/99-app-fonts.conf

RUN fc-cache -fv

COPY --from=builder /app/document-converter /app/document-converter
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

EXPOSE 5000

VOLUME [ "/app/tmp" ]

ENTRYPOINT ["/app/entrypoint.sh"]