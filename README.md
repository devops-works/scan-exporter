# Scan Exporter

Scan Exporter is tool designed to scan TCP open ports and send ICMP requests to a targets list from a [Kubernetes](https://kubernetes.io) cluster and export the results to [Prometheus](https://prometheus.io/) for monitoring.

However, Kubernetes is not necessary and you can run it locally too (see [Run it locally](#run-it-locally)) !

To fully use this tool, you will need some tools (`helm`, `kubectl`, `minikube` or `kind`, `k9s`...). If you didn't installed them yet, we have [something for you](https://github.com/devops-works/binenv).

## Installation

There is 2 ways to install scan-exporter.

You can clone this repo :

```
$ git clone https://github.com/devops-works/scan-exporter.git
```

and build the source file :

```
$ cd scan-exporter/
$ go build .
```

Or you can get the binary in [the releases](https://github.com/devops-works/scan-exporter/releases).

### Run it locally

You should check [this section](#configure-targets) to learn how to configure your targets before continuing.

```
$ ./scan-exporter [OPTIONS]
```

The differents options are :

```
OPTIONS:

-config <path/to/config/file.yaml>
    Path to config file.
    Default: config.yaml (in the current directory).

-pprof.addr <ip:port>
    pprof server address. pprof will expose it's metrics on this address.
    Default: 127.0.0.1:6060
```

**Note**: ICMP can fail if you don't have `root` permissions. However, it will not prevent scans from being realised.

### Run it in Docker

You can get the Docker image online : `devopsworks/scan-exporter`

or your can build it locally :

```
$ docker build -t <image tag> .
```

To work properly, a Redis container in the same network and with the port 6379 listening is needed.

Best practice is to create a docker-compose, else you can run both locally and bind their ports to 127.0.0.1.

**Note**: The config file is copied inside the image while creating the docker image. It is not possible to change it once the image is built.

### Run it in Kubernetes

Thanks to the Helm chart provided in the repo (deploy/helm), it is really easy to deploy the application inside a Kubernetes cluster. The following example will use Kind to create a cluster locally. If you don't have `helm`, `kubectl` or `kind`, you should try [binenv](https://github.com/devops-works/binenv) ;)

**Note**: The always-up-to-date Helm charts are [in our Helm charts repo](https://github.com/devops-works/helm-charts).

First, create the Kubernetes cluster (here, with `kind`):

```
$ kind create cluster
```

Next, use `helm` to install your chart. Before that, you have to make sure that you are using the right cluster :

```
$ kubectl config use-context kind-kind      # or whatever your context is called
Switched to context "kind-kind".
$ helm install <your release name> deploy/helm/
```

If everything went good, you should see something like this :

```
NAME: <your release name>
LAST DEPLOYED: Wed Oct 21 14:51:52 2020
NAMESPACE: default
STATUS: deployed
REVISION: 1
TEST SUITE: None
NOTES:
You just installed...

                    ▓▓▓▓▓▓
                  ██░░░░░░██
                ▒▒░░░░░░░░░░▓▓▓▓
              ▓▓░░░░░░░░░░░░░░▒▒▓▓
          ████▒▒░░░░░░░░░░░░░░░░▓▓
    ▓▓▓▓▓▓░░░░░░▒▒░░░░░░░░░░░░▒▒▓▓▓▓▓▓
  ▓▓▒▒░░░░░░░░░░░░▒▒░░░░░░░░▒▒░░░░░░▒▒▓▓
  ██▒▒▒▒░░░░░░░░▒▒▒▒▒▒▒▒▒▒▒▒░░░░░░░░░░▒▒██
    ▓▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒██
      ██▓▓▓▓▓▓██▓▓▓▓▓▓████▓▓▓▓▓▓▓▓▓▓▓▓██

          ==========================
          |scan-exporter Helm chart|
          ==========================

Your current release is named `<your release name>`.

To get more information about this release, try:

    $ helm status <your release name>
    $ helm get all <your release name>
```

To verify that everything is up in your cluster, try :

```
$ kubectl get pods
NAME                                                       READY   STATUS      RESTARTS   AGE
<your release name>-scan-exporter-chart-6d4d75dfcb-52zfp   1/1     Running     0          97s
```

## Configure targets

With Scan Exporter, there is multiple ways to configure your targets depending on how you will use it.

### Locally/Docker

You can rename config-sample.yaml which is provided in this repo to config.yaml :

```
$ mv config-sample.yaml config.yaml
```

and then use this file to describe all the targets you want to scan. The config file sample provided contains everything you can configure about a target.

You can give the path of the config file with the `-config` flag. (See [Run it locally](#run-it-locally)), both locally and or Docker.

We recommend to keep `limit: 1024`, as higher values can induce errors in scans.

### Kubernetes

If you plan to use Scan Exporter in Kubernetes, you must configure your targets in the `values.yaml` file in the Helm Chart, as `config.yaml` will be overwritten.

## Metrics exposed

The metrics exposed by Scan Exporter itself are the following:

* `scanexporter_uptime_sec`: Scan Exporter uptime, in seconds. The minimal resolution is 5 seconds. 

* `scanexporter_targets_number_total`: Number of targets detected in configuration file.

* `scanexporter_icmp_not_responding_total`: Number of targets that doesn't respond to ICMP ping requests. 

* `scanexporter_open_ports_total`: Number of ports that are open for each target.

* `scanexporter_unexpected_open_ports_total`: Number of ports that are open, and shouldn't be, for each target.

* `scanexporter_unexpected_closed_ports_total`: Number of ports that are closed, and shouldn't be, for each target.

* `scanexporter_diff_ports_total`: Number of ports that are in a different state from previous scan, for each target.

You can also fetch metrics from Go, promhttp etc...

## References

* [Prometheus](https://prometheus.io/)
* [Docker](https://docs.docker.com/)
* [Kubernetes](https://kubernetes.io)
* [Helm](https://helm.sh)
* [Kind](https://kind.sigs.k8s.io/)
* [The last binary you'll ever install](https://github.com/devops-works/binenv)

## License

[MIT](https://choosealicense.com/licenses/mit/)

## Swag zone

[![forthebadge](https://forthebadge.com/images/badges/made-with-go.svg)](https://forthebadge.com)
[![forthebadge](https://forthebadge.com/images/badges/built-with-love.svg)](https://forthebadge.com)
[![forthebadge](https://forthebadge.com/images/badges/open-source.svg)](https://forthebadge.com)
[![forthebadge](https://forthebadge.com/images/badges/powered-by-black-magic.svg)](https://forthebadge.com)
