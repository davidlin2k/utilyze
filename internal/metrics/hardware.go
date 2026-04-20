package metrics

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"os"
	"sort"
	"strings"
)

type HostIDSource string

const (
	HostIDSourceMachineID HostIDSource = "machine-id"
	HostIDSourceGPUs      HostIDSource = "gpu-fingerprint"
	HostIDSourceEphemeral HostIDSource = "ephemeral-uuid"
)

func GenerateHostID(gpuUUIDs []string) string {
	if mid, err := readMachineID(); err == nil {
		slog.Debug("host identity", "source", HostIDSourceMachineID)
		return sha256Hex("mid:" + mid)[:24]
	}

	if len(gpuUUIDs) > 0 {
		sorted := append([]string(nil), gpuUUIDs...)
		sort.Strings(sorted)
		slog.Debug("host identity", "source", HostIDSourceGPUs)
		return sha256Hex("gpus:" + strings.Join(sorted, "|"))[:24]
	}

	var b [16]byte
	_, _ = rand.Read(b[:])
	slog.Debug("host identity", "source", HostIDSourceEphemeral)
	return sha256Hex("rand:" + hex.EncodeToString(b[:]))[:24]
}

func GenerateGpuID(gpuUUID string) string {
	return sha256Hex(gpuUUID)[:12]
}

func readMachineID() (string, error) {
	data, err := os.ReadFile("/etc/machine-id")
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(data))
	if id == "" {
		return "", os.ErrNotExist
	}
	return id, nil
}

func sha256Hex(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
