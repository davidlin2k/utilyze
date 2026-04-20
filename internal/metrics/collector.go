package metrics

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/systalyze/utilyze/internal/ffi/nvml"
	"github.com/systalyze/utilyze/internal/ffi/sampler"
)

type Collector struct {
	nv              *nvml.Client
	sampler         *sampler.Sampler
	metricsInterval time.Duration
	deviceIds       []int
}

func NewCollector(deviceIds []int, metricsInterval time.Duration) (*Collector, error) {
	// Lock this goroutine to a single OS thread during initialization only. NVML init may retain a CUDA
	// primary context that is thread-local; the sampler's BeginSession needs that context current on the
	// same thread. After init, Poll only reads a ring buffer and doesn't need any CUDA context.
	return withLockedOSThread(func() (*Collector, error) {
		nv, err := nvml.Init()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize NVML: %w", err)
		}

		if len(deviceIds) == 0 {
			numDevices, err := nv.GetDeviceCount()
			if err != nil {
				return nil, fmt.Errorf("failed to get device count: %w", err)
			}
			deviceIds = make([]int, numDevices)
			for i := 0; i < numDevices; i++ {
				deviceIds[i] = i
			}
		}

		s, err := sampler.Init(deviceIds, sampler.DefaultMetrics, metricsInterval)
		if err != nil {
			return nil, err
		}
		return &Collector{
			nv:              nv,
			sampler:         s,
			metricsInterval: metricsInterval,
			deviceIds:       s.InitializedDeviceIDs(),
		}, nil
	})
}

type DeviceSnapshot struct {
	DeviceID      int
	ComputeSOLPct float64
	MemorySOLPct  float64
}

type BandwidthSnapshot struct {
	DeviceID               int
	PCIeTxBytesPerSecond   float64
	PCIeRxBytesPerSecond   float64
	NVLinkTxBytesPerSecond float64
	NVLinkRxBytesPerSecond float64
}

type MetricsSnapshot struct {
	Timestamp          time.Time
	DeviceSnapshots    []DeviceSnapshot
	BandwidthSnapshots []BandwidthSnapshot
}

func (c *Collector) Start(ctx context.Context, metrics chan MetricsSnapshot) {
	defer close(metrics)
	t := time.NewTicker(c.metricsInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			pollTime := time.Now()
			deviceSnapshots := make([]DeviceSnapshot, 0, len(c.deviceIds))
			bandwidthSnapshots := make([]BandwidthSnapshot, 0, len(c.deviceIds))
			for _, deviceID := range c.deviceIds {
				snapshot, err := c.sampler.Poll(deviceID)
				if err == nil {
					deviceSnapshots = append(deviceSnapshots, DeviceSnapshot{
						DeviceID:      deviceID,
						ComputeSOLPct: snapshot.ComputeSOLPct,
						MemorySOLPct:  snapshot.MemorySOLPct,
					})
				}

				bandwidthSnapshot, err := c.nv.PollBandwidth(deviceID, pollTime)
				if err != nil {
					continue
				}
				bandwidthSnapshots = append(bandwidthSnapshots, BandwidthSnapshot{
					DeviceID:               deviceID,
					PCIeTxBytesPerSecond:   bandwidthSnapshot.PCIeTxBps,
					PCIeRxBytesPerSecond:   bandwidthSnapshot.PCIeRxBps,
					NVLinkTxBytesPerSecond: bandwidthSnapshot.NVLinkTxBps,
					NVLinkRxBytesPerSecond: bandwidthSnapshot.NVLinkRxBps,
				})
			}

			if len(deviceSnapshots) > 0 || len(bandwidthSnapshots) > 0 {
				metrics <- MetricsSnapshot{
					Timestamp:          pollTime,
					DeviceSnapshots:    deviceSnapshots,
					BandwidthSnapshots: bandwidthSnapshots,
				}
			}
		}
	}
}

func (c *Collector) MonitoredDeviceIDs() []int {
	return c.deviceIds
}

func (c *Collector) NVMLClient() *nvml.Client {
	return c.nv
}

func (c *Collector) Close() error {
	return c.sampler.Close()
}

func withLockedOSThread[T any](fn func() (T, error)) (T, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	return fn()
}
