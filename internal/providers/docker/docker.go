package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

// UpdateContainerEnv locates a container by name and updates an environment variable.
// Because Docker doesn't support changing environment variables on the fly,
// this function will recreate the container with the new environment variable.
func UpdateContainerEnv(containerName, key, value string) error {
	ctx := context.Background()

	// Initialize Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	// Inspect the container to get its configuration by name
	inspectJSON, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to inspect container '%s': %w", containerName, err)
	}

	containerID := inspectJSON.ID

	// Update the environment variables
	envFound := false
	var newEnv []string
	prefix := key + "="
	for _, envVar := range inspectJSON.Config.Env {
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
	inspectJSON.Config.Env = newEnv

	// Stop the old container
	stopOptions := container.StopOptions{}
	err = cli.ContainerStop(ctx, containerID, stopOptions)
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Rename the old container to avoid name collision
	oldName := strings.TrimPrefix(inspectJSON.Name, "/")
	tempName := oldName + "_old_rotate"
	err = cli.ContainerRename(ctx, containerID, tempName)
	if err != nil {
		return fmt.Errorf("failed to rename old container: %w", err)
	}

	// Create a new container with the updated configuration
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: inspectJSON.NetworkSettings.Networks,
	}

	createResp, err := cli.ContainerCreate(
		ctx,
		inspectJSON.Config,
		inspectJSON.HostConfig,
		networkConfig,
		nil,
		oldName, // Use the original name
	)

	if err != nil {
		// Try to revert rename if creation fails
		_ = cli.ContainerRename(ctx, containerID, oldName)
		return fmt.Errorf("failed to create new container: %w", err)
	}

	// Start the new container
	err = cli.ContainerStart(ctx, createResp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start new container: %w", err)
	}

	// Remove the old container
	err = cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
	if err != nil {
		return fmt.Errorf("failed to remove old container (ID: %s): %w", containerID, err)
	}

	return nil
}
