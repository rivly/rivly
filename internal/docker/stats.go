package docker

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type Stats struct {
	CPUPercent float64
	MemUsage   uint64
	MemLimit   uint64
	MemPercent float64
	NetRx      uint64
	NetTx      uint64
	BlockRead  uint64
	BlockWrite uint64
	Pids       uint64
}

func (m *Manager) ContainerStats(ctx context.Context, id int64, host, containerID string) (<-chan Stats, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return nil, err
	}
	res, err := cli.ContainerStats(ctx, containerID, client.ContainerStatsOptions{Stream: true})
	if err != nil {
		return nil, err
	}

	out := make(chan Stats)
	go func() {
		defer close(out)
		defer func() { _ = res.Body.Close() }()
		decoder := json.NewDecoder(res.Body)
		for {
			var raw container.StatsResponse
			if derr := decoder.Decode(&raw); derr != nil {
				return
			}
			select {
			case out <- computeStats(raw):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

func computeStats(s container.StatsResponse) Stats {
	cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage) - float64(s.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(s.CPUStats.SystemUsage) - float64(s.PreCPUStats.SystemUsage)
	onlineCPUs := float64(s.CPUStats.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = float64(len(s.CPUStats.CPUUsage.PercpuUsage))
	}
	cpuPercent := 0.0
	if sysDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / sysDelta) * onlineCPUs * 100
	}

	memUsed := s.MemoryStats.Usage
	if inactive, ok := s.MemoryStats.Stats["inactive_file"]; ok && inactive <= memUsed {
		memUsed -= inactive
	} else if cache, ok := s.MemoryStats.Stats["cache"]; ok && cache <= memUsed {
		memUsed -= cache
	}
	memPercent := 0.0
	if s.MemoryStats.Limit > 0 {
		memPercent = float64(memUsed) / float64(s.MemoryStats.Limit) * 100
	}

	var netRx, netTx uint64
	for _, n := range s.Networks {
		netRx += n.RxBytes
		netTx += n.TxBytes
	}
	var blockRead, blockWrite uint64
	for _, b := range s.BlkioStats.IoServiceBytesRecursive {
		switch strings.ToLower(b.Op) {
		case "read":
			blockRead += b.Value
		case "write":
			blockWrite += b.Value
		}
	}

	return Stats{
		CPUPercent: cpuPercent,
		MemUsage:   memUsed,
		MemLimit:   s.MemoryStats.Limit,
		MemPercent: memPercent,
		NetRx:      netRx,
		NetTx:      netTx,
		BlockRead:  blockRead,
		BlockWrite: blockWrite,
		Pids:       s.PidsStats.Current,
	}
}
