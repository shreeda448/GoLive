package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/moby/moby/api/types/image"
)

func RunBuild() error {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer cli.Close()
	imageName := "docker.io/library/golang:alpine"
	reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()
	io.Copy(os.Stdout, reader)

	fmt.Println("\n2. Creating Container...")
	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image: "alpine",
			Cmd:   []string{"echo", "hello world from inside docker!"},
			Tty:   false,
		},
		&container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: "~/programming/projects/goLive/learning_phase/mounted_folder",
					Target: "/compiled-binaries",
				},
			},
		},
		nil, nil, "",
	)
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}

	fmt.Println("3. Starting Container...")
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		log.Fatalf("Failed to start container: %v", err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			log.Fatalf("Container error: %v", err)
		}
	case <-statusCh:
	}

	fmt.Println("4. Container finished! Fetching logs:")
	out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		log.Fatalf("Failed to get logs: %v", err)
	}
	defer out.Close()

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
}
