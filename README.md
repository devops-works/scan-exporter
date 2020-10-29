# Scan Exporter

Scan Exporter is tool designed to scan TCP open ports and send ICMP requests to a targets list from a [Kubernetes](https://kubernetes.io) cluster and export the results to [Prometheus](https://prometheus.io/) for monitoring.

However, Kubernetes is not necessary and you can run it locally too (see [Run it locally](#run-it-locally)) !

To fully use this tool, you will need some tools (`helm`, `kubectl`, `minikube` or `kind`, `k9s`...). If you didn't installed them yet, we have [something for you](https://github.com/devops-works/binenv).

## Technical decisions

We have decided to go with Prometheus and Redis because those technologies fit our requirements. However, thanks to the `storage` and `metrics` interfaces, you are free to use whatever you want to replace them, as long as those interfaces are implemented.

Obviously, PR to implement others datastores and metrics handlers are welcomed !

## Getting started

You can clone this repo :

```
$ git clone https://github.com/devops-works/scan-exporter.git
```

and enter inside the fresh new folder :

```
$ cd scan-exporter/
```

Once inside, you should check [this section](#configure-targets) to learn how to configure your targets before continuing.

You can also get it already built from the releases.

### Run it locally

Golang is needed to build the program. See [Install Golang](https://golang.org/doc/install).

```
$ go build .
$ ./scan-exporter [OPTIONS]
```

The differents options are :

```
OPTIONS:

-config <path/to/config/file.yaml>
    Path to config file.
    Default: config.yaml (in the current directory).

-log.level {info|warn|error}
    Log level.
    Default: info

-pprof.addr <ip:port>
    pprof server address. pprof will expose it's metrics on this address.
    Default: 127.0.0.1:6060

-redis.url <url>
    Redis database URL.
    Default: redis://127.0.0.1:6379/0
```

**NOTE 1** Note that ICMP can fail if you don't have `root` permissions. However, it will not prevent other scans from being realised.

### Run it in Docker

You can get the Docker image online : `devopsworks/scan-exporter`

or your can build it locally :

```
$ docker build -t <image tag> .
```

To work properly, a Redis container in the same network and with the port 6379 listening is needed.

Best practice is to create a docker-compose, else you can run both locally and bind their ports to 127.0.0.1.

**NOTE 1** The config file is copied inside the image while creating the docker image. It is not possible to change it once the image is built.

### Run it in Kubernetes

Thanks to the Helm chart provided in the repo (deploy/helm), it is really easy to deploy the application inside a Kubernetes cluster. The following example will use Kind to create a cluster locally. If you don't have `helm`, `kubectl` or `kind`, you should try [binenv](https://github.com/devops-works/binenv) ;)

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
<your release name>-redis-master-0                         1/1     Running     0          97s
<your release name>-redis-slave-0                          1/1     Running     0          97s
<your release name>-redis-slave-1                          1/1     Running     0          58s
<your release name>-scan-exporter-chart-6c967b8847-vk5fq   1/1     Running     0          97s
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

### Kubernetes

If you plan to use Scan Exporter in Kubernetes, you definitely should configure your targets in the Helm chart.

For now, they are defined in the config-map, but it will be shortly moved to values.yaml.

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