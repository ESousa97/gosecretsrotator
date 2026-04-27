package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// UpdateContainerEnv locates a container by name and updates an environment variable.
// Because Docker doesn't support changing environment variables on the fly,
// this function will recreate the container with the new environment variable.
func UpdateContainerEnv(containerName, key, value string) error {
	ctx := context.Background()

	cli, err := client.New(client.FromEnv)
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	inspectRes, err := cli.ContainerInspect(ctx, containerName, client.ContainerInspectOptions{})
	if err != nil {
		return fmt.Errorf("failed to inspect container '%s': %w", containerName, err)
	}
	inspect := inspectRes.Container
	containerID := inspect.ID

	// Refuse to recreate compose-managed containers: doing so would orphan the
	// container from its compose project. Tell the user to inject into the
	// compose .env file instead.
	if proj, ok := inspect.Config.Labels["com.docker.compose.project"]; ok {
		return fmt.Errorf("container '%s' is managed by docker compose (project=%q); refusing to recreate it. Use `inject file` against the compose .env file and run `docker compose up -d` instead", containerName, proj)
	}

	envFound := false
	var newEnv []string
	prefix := key + "="
	for _, envVar := range inspect.Config.Env {
		if strings.HasPrefix(envVar, prefix) {
			newEnv = append(newEnv, prefix+value)
			envFound = true
		} else {
			newEnv = append(newEnv, envVar)
		}
	}
	if !envFound {
		newEnv = append(newEnv, prefix+value)
	}
	inspect.Config.Env = newEnv

	if _, err = cli.ContainerStop(ctx, containerID, client.ContainerStopOptions{}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	oldName := strings.TrimPrefix(inspect.Name, "/")
	tempName := oldName + "_old_rotate"
	if _, err = cli.ContainerRename(ctx, containerID, client.ContainerRenameOptions{NewName: tempName}); err != nil {
		return fmt.Errorf("failed to rename old container: %w", err)
	}

	var endpoints map[string]*network.EndpointSettings
	if inspect.NetworkSettings != nil {
		endpoints = inspect.NetworkSettings.Networks
	}

	createRes, err := cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:           inspect.Config,
		HostConfig:       inspect.HostConfig,
		NetworkingConfig: &network.NetworkingConfig{EndpointsConfig: endpoints},
		Name:             oldName,
	})
	if err != nil {
		_, _ = cli.ContainerRename(ctx, containerID, client.ContainerRenameOptions{NewName: oldName})
		return fmt.Errorf("failed to create new container: %w", err)
	}

	if _, err = cli.ContainerStart(ctx, createRes.ID, client.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start new container: %w", err)
	}

	if _, err = cli.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove old container (ID: %s): %w", containerID, err)
	}

	return nil
}
