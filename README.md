# Scan Exporter

Export ports scans to [Prometheus](https://prometheus.io/).

## Cool zone

[![forthebadge](https://forthebadge.com/images/badges/made-with-go.svg)](https://forthebadge.com)
[![forthebadge](https://forthebadge.com/images/badges/built-with-love.svg)](https://forthebadge.com)
[![forthebadge](https://forthebadge.com/images/badges/open-source.svg)](https://forthebadge.com)

## Installation

Clone this repository and build the binary with Go :

```
$ git clone https://github.com/devops-works/scan-exporter.git
$ cd scan-exporter
$ go build .
```

## Usage

### Basic usage

```
$ sudo ./scan-exporter
```

### Flags

Scan Exporter accepts some flags :
```
FLAGS
    - config </path/to/conf/file>
        path to config file
        default: config.yaml

    - loglevel <info|debug|warn>
        log level to use
        default: info
```
## Internals

Scan Exporter reads a YAML configuration file to know which hosts to scan :

```yaml
---
targets:
  - name: "app1"
    ip: "89.168.42.229"
    tcp:
      period: "12h"
      range: "reserved"
      expected: "22,80,443"
    udp:
      period: "1d"
      range: "all"
      expected: "22,53"
    icmp:
      period: "1m"
```

By default, `config.yaml` is expected to be at the root of the folder. But you can specify a configuration file path via the flags (see Usage).

### Targets
A target has a minimum of 3 fields : `name`, `ip`, and a protocol (`tcp`, `udp`, `icmp`) and a maximum of 5.

* `name` is for readability : it doesn't play any major role ;
* `ip` is the IP of the host you want to scan ;
* protocol (see below).
  
### Protocol

Supported protocols are `tcp`, `udp` and `icmp`. For every protocol, you have to specify a scanning period. For `tcp` and `udp`, you have to add a `range` of ports to scan, and `expected`, which will hold which ports are supposed to be opened.

### Period

`period` is the period between each scan. Authorized `period` values are any number followed by `s`, `m`, `h` or `d` (respectively seconds, minutes, hours and days).

### Range and expected

* `range` is the ports range to scan ;
* `expected` is ports that should be opened.

Authorized `range` and `expected` values are any number between 0 and 65535 separated by a coma or a dash. It is possible to mix dashes and comas : `22,80-443,9001` will work.

## Prerequisites

* Go
* Sudo

## License

MIT