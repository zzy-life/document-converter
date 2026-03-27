# Document Converter

A simple HTTP server written in Go that converts office documents using LibreOffice in headless mode. Listens on port `5000` and automatically cleans up temporary files older than one hour.

## Features

- Convert Word documents (`.doc`, `.docx`, `.odt`, `.rtf`) to PDF
- Convert Excel spreadsheets (`.xls`, `.xlsx`, `.ods`, `.csv`) to PDF
- Convert PowerPoint presentations (`.ppt`, `.pptx`, `.odp`) to PDF
- Convert `.doc` / `.html` to `.docx`
- Mountable custom fonts via Docker volume
- Automatic cleanup of temporary files after one hour

## Requirements

- **Go**: [Download Go](https://golang.org/dl/)
- **LibreOffice**: Must be installed and accessible via the `soffice` command

## API

### `POST /convert/to-pdf`

Convert an Office document to PDF. Supported formats:

| Format | Extensions |
|--------|------------|
| Word   | `.doc` `.docx` `.odt` `.rtf` |
| Excel  | `.xls` `.xlsx` `.ods` `.csv` |
| PowerPoint | `.ppt` `.pptx` `.odp` |

```bash
# Word
curl -X POST -F "file=@example.docx" http://localhost:5000/convert/to-pdf -o output.pdf

# Excel
curl -X POST -F "file=@example.xlsx" http://localhost:5000/convert/to-pdf -o output.pdf

# PowerPoint
curl -X POST -F "file=@example.pptx" http://localhost:5000/convert/to-pdf -o output.pdf
```

**Response**: PDF file (`application/pdf`)

---

### `POST /convert/to-docx`

Convert a `.doc` or `.html` file to `.docx`.

```bash
# DOC to DOCX
curl -X POST -F "file=@example.doc" http://localhost:5000/convert/to-docx -o output.docx

# HTML to DOCX
curl -X POST -F "file=@example.html" http://localhost:5000/convert/to-docx -o output.docx
```

**Response**: DOCX file (`application/vnd.openxmlformats-officedocument.wordprocessingml.document`)

---

### `GET /`

Health check. Returns `OK`.

## Docker

### docker run

```bash
docker run -d \
  --name document-converter \
  -p 5000:5000 \
  -v $(pwd)/fonts:/app/fonts:ro \
  -v $(pwd)/tmp:/app/tmp \
  -e LANG=zh_CN.UTF-8 \
  -e LC_ALL=zh_CN.UTF-8 \
  zzy1998/document-converter:latest
```

字体挂载到 `/app/fonts` 后容器启动时自动加载，无需再手动 `docker exec` 复制字体或刷新缓存。

### Docker Compose

```yaml
services:
  document-converter:
    image: zzy1998/document-converter:latest
    ports:
      - "5000:5000"
    restart: unless-stopped
    environment:
      - LANG=zh_CN.UTF-8
      - LC_ALL=zh_CN.UTF-8
    volumes:
      - ./fonts:/app/fonts:ro
      - ./tmp:/app/tmp
```

## Local Build

```bash
go build -o document-converter .
./document-converter
```

## License

MIT License. See `LICENSE` for details.
