package services

import (
	"context"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

type MetricSample struct {
	CapturedAt        time.Time `json:"capturedAt"`
	HeapUsedBytes     int64     `json:"heapUsedBytes"`
	HeapMaxBytes      int64     `json:"heapMaxBytes"`
	SystemMemoryTotal int64     `json:"systemMemoryTotalBytes"`
	SystemMemoryUsed  int64     `json:"systemMemoryUsedBytes"`
	DiskTotalBytes    int64     `json:"diskTotalBytes"`
	DiskUsedBytes     int64     `json:"diskUsedBytes"`
	ProcessCpuLoad    float64   `json:"processCpuLoad"`
	SystemCpuLoad     float64   `json:"systemCpuLoad"`
}

func CaptureMetrics(db *sqlx.DB, diskPath string) (MetricSample, error) {
	proc, _ := process.NewProcess(int32(os.Getpid()))
	memStat, _ := mem.VirtualMemory()
	diskStat, err := disk.Usage(diskPath)
	if err != nil {
		diskStat, _ = disk.Usage("/")
	}
	processRSS := int64(0)
	processCPU := float64(0)
	if proc != nil {
		rss, _ := proc.MemoryInfo()
		if rss != nil {
			processRSS = int64(rss.RSS)
		}
		cpuPerc, _ := proc.CPUPercent()
		processCPU = cpuPerc / 100.0
	}
	sysCPU, _ := cpu.Percent(0, false)
	sysCPUValue := 0.0
	if len(sysCPU) > 0 {
		sysCPUValue = sysCPU[0] / 100.0
	}
	sample := MetricSample{
		CapturedAt:        time.Now().UTC(),
		HeapUsedBytes:     processRSS,
		HeapMaxBytes:      int64(memStat.Total),
		SystemMemoryTotal: int64(memStat.Total),
		SystemMemoryUsed:  int64(memStat.Total - memStat.Available),
		DiskTotalBytes:    int64(diskStat.Total),
		DiskUsedBytes:     int64(diskStat.Used),
		ProcessCpuLoad:    processCPU,
		SystemCpuLoad:     sysCPUValue,
	}

	_, err = db.Exec(`
INSERT INTO server_metric_samples (
  id, captured_at, heap_used_bytes, heap_max_bytes, system_memory_total_bytes,
  system_memory_used_bytes, disk_total_bytes, disk_used_bytes, process_cpu_load, system_cpu_load
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
`, uuid.NewString(), sample.CapturedAt, sample.HeapUsedBytes, sample.HeapMaxBytes, sample.SystemMemoryTotal,
		sample.SystemMemoryUsed, sample.DiskTotalBytes, sample.DiskUsedBytes, sample.ProcessCpuLoad, sample.SystemCpuLoad)
	if err != nil {
		return MetricSample{}, err
	}
	return sample, nil
}

func LatestMetrics(db *sqlx.DB, limit int) ([]MetricSample, error) {
	type row struct {
		CapturedAt        time.Time `db:"captured_at"`
		HeapUsedBytes     int64     `db:"heap_used_bytes"`
		HeapMaxBytes      int64     `db:"heap_max_bytes"`
		SystemMemoryTotal int64     `db:"system_memory_total_bytes"`
		SystemMemoryUsed  int64     `db:"system_memory_used_bytes"`
		DiskTotalBytes    int64     `db:"disk_total_bytes"`
		DiskUsedBytes     int64     `db:"disk_used_bytes"`
		ProcessCpuLoad    float64   `db:"process_cpu_load"`
		SystemCpuLoad     float64   `db:"system_cpu_load"`
	}
	rows := []row{}
	if err := db.Select(&rows, `
SELECT captured_at, heap_used_bytes, heap_max_bytes, system_memory_total_bytes,
       system_memory_used_bytes, disk_total_bytes, disk_used_bytes, process_cpu_load, system_cpu_load
FROM server_metric_samples
ORDER BY captured_at DESC
LIMIT $1
`, limit); err != nil {
		return nil, err
	}
	items := make([]MetricSample, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		items = append(items, MetricSample{
			CapturedAt:        rows[i].CapturedAt,
			HeapUsedBytes:     rows[i].HeapUsedBytes,
			HeapMaxBytes:      rows[i].HeapMaxBytes,
			SystemMemoryTotal: rows[i].SystemMemoryTotal,
			SystemMemoryUsed:  rows[i].SystemMemoryUsed,
			DiskTotalBytes:    rows[i].DiskTotalBytes,
			DiskUsedBytes:     rows[i].DiskUsedBytes,
			ProcessCpuLoad:    rows[i].ProcessCpuLoad,
			SystemCpuLoad:     rows[i].SystemCpuLoad,
		})
	}
	return items, nil
}

type MetricsHub struct {
	clients map[*websocket.Conn]bool
	ch      chan MetricSample
}

func NewMetricsHub() *MetricsHub {
	return &MetricsHub{
		clients: map[*websocket.Conn]bool{},
		ch:      make(chan MetricSample, 16),
	}
}

func (h *MetricsHub) Run(ctx context.Context) {
	for {
		select {
		case sample := <-h.ch:
			for conn := range h.clients {
				_ = conn.WriteJSON(sample)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *MetricsHub) Broadcast(sample MetricSample) {
	select {
	case h.ch <- sample:
	default:
	}
}

func (h *MetricsHub) Add(conn *websocket.Conn) {
	h.clients[conn] = true
}

func (h *MetricsHub) Remove(conn *websocket.Conn) {
	delete(h.clients, conn)
}
