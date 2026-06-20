package executor

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

func RunBuild(repoURL string) error {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %v", err)
	}
	defer cli.Close()

	imageName := "docker.io/library/golang:alpine"
	reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %v", err)
	}
	defer reader.Close()
	io.Copy(os.Stdout, reader)
	base := path.Base(repoURL)
	repoName := strings.TrimSuffix(base, ".git")
	branchName := "master"
	dirOfSourceCode := fmt.Sprintf("./%s-%s", repoName, branchName)
	buildCmd := fmt.Sprintf("cd %s && go build -o /workspace/compiled-binary ./cmd/api", dirOfSourceCode)
	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image:      "golang:alpine",
			WorkingDir: "/workspace",
			Cmd:        []string{"sh", "-c", buildCmd},
		},
		nil, nil, nil, "",
	)
	if err != nil {
		return fmt.Errorf("failed to create container: %v", err)
	}
	cleanRepoURL := strings.TrimSuffix(repoURL, ".git")
	suffix := "archive/refs/heads/master.tar.gz"
	finalURL := cleanRepoURL + "/" + path.Clean(suffix)
	response, err := http.Get(finalURL)
	if err != nil {
		return fmt.Errorf("failed to fetch source code: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download repo")
	}
	defer response.Body.Close()
	cli.CopyToContainer(ctx, resp.ID, "/workspace", response.Body, container.CopyToContainerOptions{})
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("container error: %v", err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err == nil {
		stdcopy.StdCopy(os.Stdout, os.Stderr, out)
		out.Close()
	}

	containerReader, _, err := cli.CopyFromContainer(ctx, resp.ID, "/workspace/compiled-binary")
	if err != nil {
		return fmt.Errorf("failed to copy from the container: %v", err)
	}
	defer containerReader.Close()
	tarReader := tar.NewReader(containerReader)
	_, err = tarReader.Next()
	if err != nil {
		return fmt.Errorf("some error occured: %v", err)
	}
	fileReader, err := os.Create("final-build-output")
	if err != nil {
		return fmt.Errorf("failed to create file:%v", err)
	}
	defer fileReader.Close()
	io.Copy(fileReader, tarReader)

	cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})

	return nil
}
