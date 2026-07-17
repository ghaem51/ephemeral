package docker

import (
	"fmt"
	"net/netip"
	"strconv"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

func createOptions(spec domain.EnvironmentSpec) client.ContainerCreateOptions {
	port := network.MustParsePort(fmt.Sprintf("%d/tcp", spec.ContainerPort))
	return client.ContainerCreateOptions{
		Name: containerName(spec),
		Config: &container.Config{
			Image: spec.Image,
			Env:   []string{"ENVIRONMENT_NAME=" + spec.Name},
			Labels: map[string]string{
				LabelManaged: "true", LabelEnvironmentID: spec.ID, LabelEnvironmentName: spec.Name,
			},
			ExposedPorts: network.PortSet{port: struct{}{}},
		},
		HostConfig: &container.HostConfig{
			Privileged: false,
			PortBindings: network.PortMap{
				port: []network.PortBinding{{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: ""}},
			},
		},
	}
}

func runtimeFromInspection(
	containerID string,
	containerPort int,
	inspection client.ContainerInspectResult,
) (domain.RuntimeInfo, error) {
	if inspection.Container.NetworkSettings == nil {
		return domain.RuntimeInfo{ContainerID: containerID, ContainerPort: containerPort},
			fmt.Errorf("inspect Docker container %q: network settings are missing", containerID)
	}
	port := network.MustParsePort(fmt.Sprintf("%d/tcp", containerPort))
	bindings := inspection.Container.NetworkSettings.Ports[port]
	if len(bindings) == 0 || bindings[0].HostPort == "" {
		return domain.RuntimeInfo{ContainerID: containerID, ContainerPort: containerPort},
			fmt.Errorf("inspect Docker container %q: host port for %s is missing", containerID, port)
	}
	hostPort, err := strconv.Atoi(bindings[0].HostPort)
	if err != nil || hostPort < 1 || hostPort > 65535 {
		return domain.RuntimeInfo{ContainerID: containerID, ContainerPort: containerPort},
			fmt.Errorf("inspect Docker container %q: invalid host port %q", containerID, bindings[0].HostPort)
	}
	return domain.RuntimeInfo{
		ContainerID: containerID, ContainerPort: containerPort,
		HostPort: hostPort, URL: localURL(hostPort),
	}, nil
}
