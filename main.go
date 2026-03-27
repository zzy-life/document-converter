package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const tempDir = "./tmp" // Directory for temporary files

func main() {
	// Ensure the temporary directory exists
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		fmt.Println("Failed to create temp directory:", err)
		return
	}

	// Start the file cleanup goroutine
	go cleanupOldFiles(tempDir, 1*time.Hour)

	http.HandleFunc("/", handleHealthCheck)
	http.HandleFunc("/convert/to-pdf", handleConvert)
	http.HandleFunc("/convert/to-docx", handleDocToDocx)
	http.HandleFunc("/convert/local-to-docx", handleLocalHtmlToDocx)

	fmt.Println("Starting server on :5000")
	if err := http.ListenAndServe(":5000", nil); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

// pdfFilter returns the LibreOffice PDF export filter for the given file extension.
func pdfFilter(ext string) (string, bool) {
	switch ext {
	case ".doc", ".docx", ".odt", ".rtf":
		return "writer_pdf_Export", true
	case ".xls", ".xlsx", ".ods", ".csv":
		return "calc_pdf_Export", true
	case ".ppt", ".pptx", ".odp":
		return "impress_pdf_Export", true
	default:
		return "", false
	}
}

func handleConvert(w http.ResponseWriter, r *http.Request) {
	// Ensure the request method is POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read uploaded file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	filter, ok := pdfFilter(ext)
	if !ok {
		http.Error(w, "Unsupported file type: "+ext, http.StatusBadRequest)
		return
	}

	baseName := time.Now().Format("20060102150405")
	inputFilePath := filepath.Join(tempDir, baseName+ext)
	outputFilePath := filepath.Join(tempDir, baseName+".pdf")

	inputFile, err := os.Create(inputFilePath)
	if err != nil {
		http.Error(w, "Failed to create temporary file", http.StatusInternalServerError)
		return
	}
	defer inputFile.Close()
	defer os.Remove(inputFilePath)

	if _, err = io.Copy(inputFile, file); err != nil {
		http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}

	// Convert to PDF using LibreOffice
	cmd := exec.Command("soffice", "--headless", "--convert-to", "pdf:"+filter, inputFilePath, "--outdir", tempDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Println(string(output))
		http.Error(w, "Failed to convert file to PDF", http.StatusInternalServerError)
		return
	}
	defer os.Remove(outputFilePath)

	pdfFile, err := os.Open(outputFilePath)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to read converted PDF", http.StatusInternalServerError)
		return
	}
	defer pdfFile.Close()

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="output.pdf"`)

	if _, err := io.Copy(w, pdfFile); err != nil {
		http.Error(w, "Failed to write PDF to response", http.StatusInternalServerError)
		return
	}
}

func handleDocToDocx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read uploaded file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".doc" && ext != ".html" && ext != ".htm" {
		http.Error(w, "Only .doc, .html, .htm files are supported", http.StatusBadRequest)
		return
	}

	baseName := time.Now().Format("20060102150405")
	inputFilePath := filepath.Join(tempDir, baseName+ext)
	outputFilePath := filepath.Join(tempDir, baseName+".docx")

	inputFile, err := os.Create(inputFilePath)
	if err != nil {
		http.Error(w, "Failed to create temporary file", http.StatusInternalServerError)
		return
	}
	defer inputFile.Close()
	defer os.Remove(inputFilePath)

	if ext == ".html" || ext == ".htm" {
		raw, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Failed to read uploaded file", http.StatusInternalServerError)
			return
		}
		fixed := fixFontNames(string(raw))
		if _, err = inputFile.WriteString(fixed); err != nil {
			http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
			return
		}
	} else if _, err = io.Copy(inputFile, file); err != nil {
		http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}

	filter := "docx:MS Word 2007 XML"
	if ext == ".html" || ext == ".htm" {
		filter = "docx:MS Word 2007 XML:EmbedImages"
	}
	cmd := exec.Command("soffice", "--headless", "--convert-to", filter, inputFilePath, "--outdir", tempDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Println(string(output))
		http.Error(w, "Failed to convert file to DOCX", http.StatusInternalServerError)
		return
	}
	defer os.Remove(outputFilePath)

	docxFile, err := os.Open(outputFilePath)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to read converted DOCX", http.StatusInternalServerError)
		return
	}
	defer docxFile.Close()

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", `attachment; filename="output.docx"`)

	if _, err := io.Copy(w, docxFile); err != nil {
		http.Error(w, "Failed to write DOCX to response", http.StatusInternalServerError)
		return
	}
}

// handleLocalHtmlToDocx converts a server-side HTML file (with local assets) to DOCX.
// It accepts a "path" form field pointing to the absolute path of the HTML file.
// Images referenced by relative paths are resolved from the HTML file's directory.
func handleLocalHtmlToDocx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	htmlPath := r.FormValue("path")
	if htmlPath == "" {
		http.Error(w, "Missing form field: path", http.StatusBadRequest)
		return
	}
	htmlPath = filepath.Clean(htmlPath)

	ext := strings.ToLower(filepath.Ext(htmlPath))
	if ext != ".html" && ext != ".htm" {
		http.Error(w, "path must point to an .html or .htm file", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(htmlPath)
	if err != nil || info.IsDir() {
		http.Error(w, "File not found: "+htmlPath, http.StatusBadRequest)
		return
	}

	content, err := os.ReadFile(htmlPath)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Fix Windows-style backslashes in src/href attributes so LibreOffice can resolve them on Linux
	fixedContent := fixFontNames(fixBackslashPaths(string(content)))

	htmlDir := filepath.Dir(htmlPath)
	baseName := time.Now().Format("20060102150405")
	tmpHtmlPath := filepath.Join(htmlDir, baseName+ext)
	outputFilePath := filepath.Join(tempDir, baseName+".docx")

	if err := os.WriteFile(tmpHtmlPath, []byte(fixedContent), 0644); err != nil {
		http.Error(w, "Failed to write temporary HTML", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpHtmlPath)

	cmd := exec.Command("soffice", "--headless", "--convert-to", "docx:MS Word 2007 XML:EmbedImages", tmpHtmlPath, "--outdir", tempDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Println(string(output))
		http.Error(w, "Failed to convert file to DOCX", http.StatusInternalServerError)
		return
	}
	defer os.Remove(outputFilePath)

	docxFile, err := os.Open(outputFilePath)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to read converted DOCX", http.StatusInternalServerError)
		return
	}
	defer docxFile.Close()

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", `attachment; filename="output.docx"`)

	if _, err := io.Copy(w, docxFile); err != nil {
		http.Error(w, "Failed to write DOCX to response", http.StatusInternalServerError)
		return
	}
}

// fixBackslashPaths replaces backslashes with forward slashes in HTML src/href attribute values.
var backslashAttrRe = regexp.MustCompile(`(?i)(src|href)="([^"]*)"`)

func fixBackslashPaths(html string) string {
	return backslashAttrRe.ReplaceAllStringFunc(html, func(match string) string {
		parts := backslashAttrRe.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		return parts[1] + `="` + strings.ReplaceAll(parts[2], `\`, `/`) + `"`
	})
}

// fixFontNames replaces English font names that LibreOffice on Linux cannot resolve
// with their Chinese equivalents which are correctly recognised.
func fixFontNames(html string) string {
	html = strings.ReplaceAll(html, "Microsoft YaHei", "微软雅黑")
	html = strings.ReplaceAll(html, "microsoft yahei", "微软雅黑")
	return html
}

// cleanupOldFiles removes files older than the specified duration from the given directory
func cleanupOldFiles(dir string, maxAge time.Duration) {
	for {
		time.Sleep(1 * time.Hour) // Check every minute

		files, err := os.ReadDir(dir)
		if err != nil {
			fmt.Println("Failed to read temp directory:", err)
			continue
		}

		for _, file := range files {
			filePath := filepath.Join(dir, file.Name())
			info, err := os.Stat(filePath)
			if err != nil {
				fmt.Println("Failed to get file info:", err)
				continue
			}

			// Check if the file is older than maxAge
			if time.Since(info.ModTime()) > maxAge {
				if err := os.Remove(filePath); err != nil {
					fmt.Println("Failed to delete file:", err)
				} else {
					fmt.Println("Deleted old file:", filePath)
				}
			}
		}
	}
}
