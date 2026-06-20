package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDeployHandler(t *testing.T) {
	tempDir := t.TempDir()
	file1 := "main.go"
	file2 := "go.mod"
	f1 := filepath.Join(tempDir, file1)
	f2 := filepath.Join(tempDir, file2)
	os.WriteFile(f1, []byte("package main; func main() {}"), 0o644)
	os.WriteFile(f2, []byte("module fakeapp \n go 1.21"), 0o644)
	jsonBody := []byte(`{"output_dir": "` + tempDir + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/deploy", bytes.NewBuffer(jsonBody))
	responseRecorder := httptest.NewRecorder()
	deployHandler(responseRecorder, req, myAsyncQ)
	if responseRecorder.Code != http.StatusAccepted {
		t.Errorf("expected status %d,got %d", http.StatusAccepted, responseRecorder.Code)
	}
	binaryPath := filepath.Join(tempDir, "compiled-binary")
	info, err := os.Stat(binaryPath)
	if os.IsNotExist(err) || info.IsDir() {
		t.Errorf("expected a compiled binary FILE, but it failed or was a directory")
	}
}
