package container

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

// Manager wraps the Docker Engine API for container lifecycle operations.
type Manager struct {
	cli *client.Client
}

// ContainerInfo is a simplified container representation for API responses.
type ContainerInfo struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Image      string            `json:"image"`
	Status     string            `json:"status"`
	State      string            `json:"state"`
	Ports      []PortBinding     `json:"ports"`
	Labels     map[string]string `json:"labels"`
	CreatedAt  time.Time         `json:"created_at"`
	CPUPercent float64           `json:"cpu_percent"`
	MemUsageMB float64           `json:"mem_usage_mb"`
	MemLimitMB float64           `json:"mem_limit_mb"`
	NetRxMB    float64           `json:"net_rx_mb"`
	NetTxMB    float64           `json:"net_tx_mb"`
}

type PortBinding struct {
	HostPort      string `json:"host_port"`
	ContainerPort string `json:"container_port"`
	Protocol      string `json:"protocol"`
}

type ResourceLimits struct {
	CPUQuota int64 `json:"cpu_quota"` // microseconds per 100ms period
	MemoryMB int64 `json:"memory_mb"` // MB
}

func NewManager(dockerHost string) (*Manager, error) {
	opts := []client.Opt{
		client.WithAPIVersionNegotiation(),
	}
	if dockerHost != "" {
		opts = append(opts, client.WithHost(dockerHost))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, err
	}

	return &Manager{cli: cli}, nil
}

func (m *Manager) Close() {
	m.cli.Close()
}

// List returns all containers (running and stopped).
func (m *Manager) List(ctx context.Context, onlyRunning bool) ([]ContainerInfo, error) {
	containers, err := m.cli.ContainerList(ctx, dockercontainer.ListOptions{
		All: !onlyRunning,
	})
	if err != nil {
		return nil, err
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")
		info := ContainerInfo{
			ID:        c.ID[:12],
			Name:      name,
			Image:     c.Image,
			Status:    c.Status,
			State:     c.State,
			Ports:     parsePorts(c.Ports),
			Labels:    c.Labels,
			CreatedAt: time.Unix(c.Created, 0),
		}
		result = append(result, info)
	}
	return result, nil
}

// Inspect returns full details of a single container by name or ID.
func (m *Manager) Inspect(ctx context.Context, nameOrID string) (types.ContainerJSON, error) {
	return m.cli.ContainerInspect(ctx, nameOrID)
}

// Start starts a stopped container.
func (m *Manager) Start(ctx context.Context, nameOrID string) error {
	return m.cli.ContainerStart(ctx, nameOrID, dockercontainer.StartOptions{})
}

// Stop gracefully stops a running container.
func (m *Manager) Stop(ctx context.Context, nameOrID string, timeout *int) error {
	opts := dockercontainer.StopOptions{}
	if timeout != nil {
		opts.Timeout = timeout
	}
	return m.cli.ContainerStop(ctx, nameOrID, opts)
}

// Remove removes a container. Force=true kills it first if running.
func (m *Manager) Remove(ctx context.Context, nameOrID string, force bool) error {
	return m.cli.ContainerRemove(ctx, nameOrID, dockercontainer.RemoveOptions{Force: force})
}

// Logs streams container logs. Returns a ReadCloser; caller is responsible for closing.
func (m *Manager) Logs(ctx context.Context, nameOrID string, tail string, follow bool) (io.ReadCloser, error) {
	return m.cli.ContainerLogs(ctx, nameOrID, dockercontainer.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Follow:     follow,
		Timestamps: true,
	})
}

// Stats returns a snapshot of resource usage for a container.
func (m *Manager) Stats(ctx context.Context, nameOrID string) (*ContainerInfo, error) {
	resp, err := m.cli.ContainerStats(ctx, nameOrID, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw types.StatsJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	info := &ContainerInfo{
		ID:         nameOrID,
		CPUPercent: calcCPUPercent(&raw),
		MemUsageMB: bytesToMB(raw.MemoryStats.Usage),
		MemLimitMB: bytesToMB(raw.MemoryStats.Limit),
		NetRxMB:    calcNetRx(&raw),
		NetTxMB:    calcNetTx(&raw),
	}
	return info, nil
}

// StreamStats sends periodic stats snapshots to the provided channel until ctx is cancelled.
func (m *Manager) StreamStats(ctx context.Context, nameOrID string, ch chan<- ContainerInfo, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := m.Stats(ctx, nameOrID)
			if err != nil {
				log.Error().Err(err).Str("container", nameOrID).Msg("stats fetch failed")
				continue
			}
			info.Name = nameOrID
			ch <- *info
		}
	}
}

// UpdateLimits applies CPU and memory resource limits to a running container.
func (m *Manager) UpdateLimits(ctx context.Context, nameOrID string, limits ResourceLimits) error {
	update := dockercontainer.UpdateConfig{
		Resources: dockercontainer.Resources{},
	}
	if limits.CPUQuota > 0 {
		update.Resources.CPUQuota = limits.CPUQuota
		update.Resources.CPUPeriod = 100000
	}
	if limits.MemoryMB > 0 {
		update.Resources.Memory = limits.MemoryMB * 1024 * 1024
	}
	_, err := m.cli.ContainerUpdate(ctx, nameOrID, update)
	return err
}

// ListByLabel returns containers matching a specific label key=value.
func (m *Manager) ListByLabel(ctx context.Context, label string) ([]ContainerInfo, error) {
	f := filters.NewArgs()
	f.Add("label", label)
	containers, err := m.cli.ContainerList(ctx, dockercontainer.ListOptions{
		All:     true,
		Filters: f,
	})
	if err != nil {
		return nil, err
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, c := range containers {
		result = append(result, ContainerInfo{
			ID:     c.ID[:12],
			Name:   strings.TrimPrefix(c.Names[0], "/"),
			Image:  c.Image,
			Status: c.Status,
			State:  c.State,
		})
	}
	return result, nil
}

func parsePorts(ports []types.Port) []PortBinding {
	bindings := make([]PortBinding, 0, len(ports))
	for _, p := range ports {
		if p.PublicPort == 0 {
			continue
		}
		bindings = append(bindings, PortBinding{
			HostPort:      fmt.Sprintf("%d", p.PublicPort),
			ContainerPort: fmt.Sprintf("%d", p.PrivatePort),
			Protocol:      p.Type,
		})
	}
	return bindings
}

func calcCPUPercent(stats *types.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage) - float64(stats.PreCPUStats.SystemUsage)
	numCPUs := float64(stats.CPUStats.OnlineCPUs)
	if numCPUs == 0 {
		numCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}
	if systemDelta == 0 {
		return 0
	}
	return math.Round((cpuDelta/systemDelta)*numCPUs*100*100) / 100
}

func calcNetRx(stats *types.StatsJSON) float64 {
	var rx uint64
	for _, net := range stats.Networks {
		rx += net.RxBytes
	}
	return bytesToMB(rx)
}

func calcNetTx(stats *types.StatsJSON) float64 {
	var tx uint64
	for _, net := range stats.Networks {
		tx += net.TxBytes
	}
	return bytesToMB(tx)
}

func bytesToMB(b uint64) float64 {
	return math.Round(float64(b)/1024/1024*100) / 100
}
