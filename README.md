# Document Converter

A simple HTTP server written in Go that converts office documents using LibreOffice in headless mode. Listens on port `5000` and automatically cleans up temporary files older than one hour.

## Features

- Convert `.xlsx` to PDF
- Convert `.doc` to `.docx`
- Mountable custom fonts via Docker volume
- Automatic cleanup of temporary files after one hour

## Requirements

- **Go**: [Download Go](https://golang.org/dl/)
- **LibreOffice**: Must be installed and accessible via the `soffice` command

## API

### `POST /convert/to-pdf`

Convert an `.xlsx` file to PDF.

```bash
curl -X POST -F "file=@example.xlsx" http://localhost:5000/convert/to-pdf -o output.pdf
```

**Response**: PDF file (`application/pdf`)

---

### `POST /convert/to-docx`

Convert a `.doc` file to `.docx`.

```bash
curl -X POST -F "file=@example.doc" http://localhost:5000/convert/to-docx -o output.docx
```

**Response**: DOCX file (`application/vnd.openxmlformats-officedocument.wordprocessingml.document`)

---

### `GET /`

Health check. Returns `OK`.

## Docker

### Pull and run

```bash
docker pull zzy1998/document-converter
docker run -p 5000:5000 zzy1998/document-converter
```

### Docker Compose

```yaml
services:
  document-converter:
    image: zzy1998/document-converter:latest
    ports:
      - "5000:5000"
    restart: unless-stopped
    volumes:
      - ./fonts:/app/fonts:ro  # optional: mount custom fonts
```

### Custom Fonts

Place `.ttf` or `.otf` font files in a local `fonts/` directory and mount it to `/app/fonts`. The container loads them automatically on startup — no manual `exec` needed.

## Local Build

```bash
go build -o document-converter .
./document-converter
```

## License

MIT License. See `LICENSE` for details.
