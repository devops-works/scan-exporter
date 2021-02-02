# Scan Exporter

Massive TCP/ICMP port scanning tool with exporter for Prometheus.

<a href="https://github.com/MariaLetta/free-gophers-pack"><img align="center" src="./img/gobug.png" width="250" height="250"/></a>

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Getting started](#getting-started)
  - [From the sources](#from-the-sources)
  - [From the releases](#from-the-releases)
  - [From `binenv`](#from-binenv)
- [Usage](#usage)
  - [CLI](#cli)
  - [Kubernetes](#kubernetes)
- [Configuration](#configuration)
  - [Configuration file](#configuration-file)
    - [`target_config`](#target_config)
    - [`tcp_config`](#tcp_config)
    - [`icmp_config`](#icmp_config)
  - [Helm](#helm)
- [Metrics](#metrics)
- [License](#license)
- [Swag zone](#swag-zone)
  
## Getting started

There is multiple ways to get `scan-exporter`:

### From the sources

```
$ git clone https://github.com/devops-works/scan-exporter.git
$ cd scan-exporter
$ go build .
```

### From the releases

Download the latest build from [the official releases](https://github.com/devops-works/scan-exporter/releases)

### From `binenv`

If you have `binenv` installed:

```
$ binenv install scan-exporter
```

## Usage

### CLI

```
USAGE: [sudo] ./scan-exporter [OPTIONS]

OPTIONS:

-config <path/to/config/file.yaml>
    Path to config file.
    Default: config.yaml (in the current directory).

-pprof.addr <ip:port>
    pprof server address. pprof will expose it's metrics on this address.
  
-log.lvl {trace,debug,info,warn,error,fatal}
    Log level.
    Default: info
```

:bulb: ICMP can fail if you don't start `scan-exporter` with `root` permissions. However, it will not prevent ports scans from being realised.

### Kubernetes

Use the provided Helm charts to deploy the application into your application. 

:warning: Those charts might not be up to date, please use [those ones](https://github.com/devops-works/helm-charts/tree/master/scan-exporter).

To deploy with charts that are, for example, under `scan-exporter/` in the current Kubernetes context:

```
$ helm install scanexporter scan-exporter/
```

## Configuration

### Configuration file

```yaml
# Hold the timeout, in seconds, that will be used all over the program (i.e for scans).
timeout: int

# Hold the number of files that can be simultaneously opened on the host.
# It will be the number of scan goroutines. We recommand 1024 or 2048, but it depends
# on the `ulimit` of the host.
limit: int

# The log level that will be used all over the program. Supported values:
# trace, debug, info, warn, error, fatal
[log_level: <string> | default = "info"]

# Configure targets.
targets:
  - [<target_config>]
```

#### `target_config`

```yaml
# Name of the target.
# It can be whatever you want.
name: <string>

# IP address of the target.
# Only IPv4 addresses are supported.
ip: <string>

# TCP scan parameters.
[tcp: <tcp_config>]

# ICMP scan parameters
[icmp: <icmp_config>]
```

#### `tcp_config`

```yaml
# TCP scan frequency. Supported values:
# {1..999}{s,m,h,d}
# For example, all those values are working and reprensent a frenquecy of
# one hour: 3600s, 60m, 1h.
period: <string>

# Range of ports to scan. Supported values:
# all, reserved, 22, 100-1000, 11,12-14,15...
range: <string>

# Ports that are expected to be open. Supported values are the same than
# for range.
expected: <string>
```

#### `icmp_config`

```yaml
# Ping frequency. Supported values are the same than for TCP's period.
period: <string>
```

Here is a working example:

```yaml
timeout: 2
limit: 1024
log_level: "warn"
targets:
  - name: "app1"
    ip: "198.51.100.42"
    tcp:
      period: "12h"
      range: "reserved"
      expected: "22,80,443"
    icmp:
      period: "1m"
  
  - name: "app2"
    ip: "198.51.100.69"
    tcp:
      period: "1d"
      range: "all"
      expected: ""
```

* `app1` will be scanned using TCP every 12 hours on all the reserved ports (1-1023), and we expect that ports 22, 80, 443 will be open, and all the others closed. It will also receive an ICMP ping every minute.
* `app2` will be scanned using TCP every day on all its ports (1-65535), and none of its ports should be open.

### Helm

The structure of the configuration file is the same, except that is should be placed inside `values.yaml`.

[See an example](https://github.com/devops-works/helm-charts/blob/master/scan-exporter/values.yaml) of `values.yaml` configuration.

## Metrics

The metrics exposed by `scan-exporter` itself are the following:

* `scanexporter_uptime_sec`: Uptime, in seconds. The minimal resolution is 5 seconds. 

* `scanexporter_targets_number_total`: Number of targets detected in configuration file.

* `scanexporter_pending_scans`: Number of scans that are in the waiting line.

* `scanexporter_icmp_not_responding_total`: Number of targets that doesn't respond to ICMP ping requests. 

* `scanexporter_open_ports_total`: Number of ports that are open for each target.

* `scanexporter_unexpected_open_ports_total`: Number of ports that are open, and shouldn't be, for each target.

* `scanexporter_unexpected_closed_ports_total`: Number of ports that are closed, and shouldn't be, for each target.

* `scanexporter_diff_ports_total`: Number of ports that are in a different state from previous scan, for each target.

You can also fetch metrics from Go, promhttp etc.

## License

[MIT](https://choosealicense.com/licenses/mit/)

Gopher from [Maria Letta](https://github.com/MariaLetta/free-gophers-pack)

## Swag zone

[![forthebadge](https://forthebadge.com/images/badges/made-with-go.svg)](https://forthebadge.com)
[![forthebadge](https://forthebadge.com/images/badges/built-with-love.svg)](https://forthebadge.com)
[![forthebadge](https://forthebadge.com/images/badges/open-source.svg)](https://forthebadge.com)
[![forthebadge](https://forthebadge.com/images/badges/powered-by-black-magic.svg)](https://forthebadge.com)
