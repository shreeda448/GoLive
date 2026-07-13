package executor

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func RunApplication(artifactURL string, port int, deployID string) error {
	resp, err := http.Get(artifactURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	binaryPath := "./running-apps"
	binaryName := deployID
	err = os.MkdirAll(binaryPath, 0o755)
	if err != nil {
		return err
	}
	finalBinaryPath := filepath.Join(binaryPath, binaryName)
	binaryFileDescriptor, err := os.Create(finalBinaryPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(binaryFileDescriptor, resp.Body)
	if err != nil {
		binaryFileDescriptor.Close()
		return err
	}
	binaryFileDescriptor.Close()
	err = os.Chmod(finalBinaryPath, 0o755)
	if err != nil {
		return err
	}
	cmd := exec.Command(finalBinaryPath)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", port))
	err = cmd.Start()
	if err != nil {
		return err
	}
	return nil
}
