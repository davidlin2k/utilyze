# Utilyze

Measure how efficiently your GPU is actually doing useful work, not just whether it's running, and shows you this live, without slowing down your workload.

![utlz in action](./assets/utlz.png)

utilyze is created by [Systalyze](https://systalyze.com).

## Requirements

- Linux amd64 (arm64 support coming soon)
- NVIDIA Ampere or newer GPU (A100, H100, H200, B200, RTX 3000+)
- CUDA Toolkit 11.0+
- `sudo` or `CAP_SYS_ADMIN` (see below), or privileged container

## Installation

```bash
curl -sSfL https://systalyze.com/utilyze/install.sh | sh
```

utilyze will likely require root for profiling capabilities depending on your host configuration (see below) and will prompt you for password during installation to install it system-wide.

If CUPTI 12+ is not found, `utlz` will prompt you to install the latest release from PyPI on first run.

## Features

```bash
# monitor all GPUs for SOL metrics
sudo utlz

# monitor specific GPUs
sudo utlz --devices 0,2

# show discovered inference server endpoints per GPU
sudo utlz --endpoints
```

Utilyze supports inference service endpoint discovery to show roofline ceilings (attainable compute speed-of-light, i.e. SOL) for each GPU. Currently only vLLM is supported, but more backend support is coming soon.

Note that a single device ID can only be monitored by a single instance of `utlz`. This is due to the way NVIDIA's Perf SDK API handles device access.

### Running without sudo

By default, NVIDIA restricts GPU profiling counters to admin users. To allow non-root access, disable the restriction on the host and reboot:

```bash
echo 'options nvidia NVreg_RestrictProfilingToAdminUsers=0' | sudo tee /etc/modprobe.d/nvidia-profiling.conf
sudo reboot
```

After this, `utlz` can run without sudo. If `utlz` warns about missing capabilities, you can disable the warning by setting `UTLZ_DISABLE_PROFILING_WARNING=1`.

### Options

- `--endpoints`: show discovered inference server endpoints per GPU (current)
- `--devices`/`UTLZ_DEVICES`: monitor specific GPUs (comma-separated list of device IDs)
- `--log`/`UTLZ_LOG`: a file to write logs to (default: no logging)
- `--log-level`/`UTLZ_LOG_LEVEL`: set the log level (default: `INFO`, other options: `DEBUG`, `WARN`, `ERROR`)
- `--version`: show the version
- `UTLZ_DISABLE_PROFILING_WARNING`: disable the warning about GPU profiling capabilities on startup
- `UTLZ_BACKEND_URL`: set the backend URL for Systalyze's roofline SOL metrics API (default: `https://api.systalyze.com/v1/utilyze`)
- `UTLZ_DISABLE_METRICS`: disable workload detection and Systalyze roofline SOL metrics API

## Build from source

To build from source you'll need to have

- Go 1.25+ for the CLI
- Docker for building the native library with wide compatibility
- CUDA Toolkit (13.1 is linked against by default but can be set via `CUDA_VERSION`)

```bash
# build the native library and the CLI
make all

# build and package the native library via Docker
make dist-tarball-docker

# build the CLI only
make utlz
```

There is experimental support for ARM64 builds using the sbsa-linux CUDA target.
