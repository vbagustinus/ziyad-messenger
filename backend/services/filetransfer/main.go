package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const StorageDir = "./storage"

type FileTransferService struct {
	storagePath string
}

func NewFileTransferService(path string) *FileTransferService {
	if err := os.MkdirAll(path, 0700); err != nil {
		log.Fatalf("Failed to create storage dir: %v", err)
	}
	return &FileTransferService{storagePath: path}
}

// UploadHandler handles file uploads and encrypts them at rest.
func (s *FileTransferService) UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileID := uuid.New().String()
	ext := filepath.Ext(header.Filename)
	savePath := filepath.Join(s.storagePath, fileID+ext+".enc")

	// Create file
	out, err := os.Create(savePath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	// Generate DEK (Data Encryption Key)
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		http.Error(w, "Crypto error", http.StatusInternalServerError)
		return
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		http.Error(w, "Crypto error", http.StatusInternalServerError)
		return
	}

	iv := make([]byte, 12) // GCM IV size
	if _, err := rand.Read(iv); err != nil {
		http.Error(w, "Crypto error", http.StatusInternalServerError)
		return
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		http.Error(w, "Crypto error", http.StatusInternalServerError)
		return
	}

	// Read content
	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Read error", http.StatusInternalServerError)
		return
	}

	// Encrypt
	ciphertext := aesGCM.Seal(nil, iv, content, nil)

	// Write IV + Ciphertext
	if _, err := out.Write(iv); err != nil {
		http.Error(w, "Write error", http.StatusInternalServerError)
		return
	}
	if _, err := out.Write(ciphertext); err != nil {
		http.Error(w, "Write error", http.StatusInternalServerError)
		return
	}

	// Return File ID and Key (Key should be protected/wrapped in real app)
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"file_id": "%s", "key": "%x"}`, fileID, key)
}

// DownloadHandler serves encrypted files.
func (s *FileTransferService) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	fileID := r.URL.Query().Get("id")
	if fileID == "" {
		http.Error(w, "Missing file ID", http.StatusBadRequest)
		return
	}

	// In a real app, strict path sanitization is needed
	matches, _ := filepath.Glob(filepath.Join(s.storagePath, fileID+"*.enc"))
	if len(matches) == 0 {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, matches[0])
}

func main() {
	svc := NewFileTransferService(StorageDir)

	http.HandleFunc("/upload", svc.UploadHandler)
	http.HandleFunc("/download", svc.DownloadHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "File Transfer Service is running")
	})

	log.Println("File Transfer Service started on :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
