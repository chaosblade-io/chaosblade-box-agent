# Chaosblade-box-agnet: Chaos Experiment Agent
![license](https://img.shields.io/github/license/chaosblade-io/chaosblade.svg)

中文版 [README](README_CN.md)

## Introduction
The Agent are mainly used for platform-side establishment, command delivery channels, and data collection functions. Therefore, if you need to perform drills on target clusters or hosts, you need to install probes on the target clusters or hosts on the end-side to organize the platform. The drill is converted into commands and sent to the target machine

## How to use
This project can be compiled and used separately, but it is more recommended to use [chaosblade-box-agent](https://github.com/chaosblade-io/chaosblade-box-agent)  tool to use. For detailed English documentation, please refer to: https://chaosblade.io/en/docs/getting-started/installation-and-deployment/agent-install/

## Compile

This project is written in golang, so you need to install the latest golang version first. The minimum supported version is 1.11. After cloning the project, enter the project directory and execute the following commands:

### Build Binary

```bash
# Build the chaos agent binary (for current platform)
make build

# Build for linux/amd64
make build_amd64

# Build for linux/arm64
make build_arm64
```

### Build Docker Image

```bash
# Build Docker image for amd64
make build_image_amd64

# Build Docker image for arm64
make build_image_arm64
```

### Build Helm Chart

```bash
# Package Helm chart for amd64
make build_chart_amd64

# Package Helm chart for arm64
make build_chart_arm64
```

### Package Agent

```bash
# Package agent binary, scripts and chaosblade tool
# Note: Requires build_amd64 or build_arm64 to be run first
make package ARCH=amd64
# or
make package ARCH=arm64
```

### Clean Build Artifacts

```bash
# Clean up build artifacts
make clean
```

### View All Available Commands

```bash
# Show help information, view all available commands
make help
```

### Code Quality

```bash
# Format Go code
make format

# Verify code formatting
make verify

# Check license headers
make license-check

# Format license headers
make license-format
```

Steps to install agent:
```bash
./chaosctl.sh install -k 015667e5361b4b0c9d42e1c10afe1d61 -p  [app-name]  -g  [app-group-name]  -P  [agent-port]  -t [chaosblade-box ip]
```

Installation example:
```bash
./chaosctl.sh install -k 015667e5361b4b0c9d42e1c10afe1d61  -p  chaos-default-app  -g  chaos-default-app-group  -P 19527 -t 127.0.0.1
```

## Bugs and Feedback
For bug report, questions and discussions please submit [GitHub Issues](https://github.com/chaosblade-io/chaosblade/issues). 

You can also contact us via:
* Dingding group (recommended for chinese): 23177705
* Gitter room: [chaosblade community] (https://gitter.im/chaosblade-io/community)
* Email: chaosblade.io.01@gmail.com
* Twitter: [chaosblade.io] (https://twitter.com/ChaosbladeI)

## Contributing
We welcome every contribution, even if it is just punctuation. See details of [CONTRIBUTING](CONTRIBUTING.md)

## License
The chaosblade-exec-os is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for the full license text.

