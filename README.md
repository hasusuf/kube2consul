# kube2consul
[![Go Report Card](https://goreportcard.com/badge/github.com/hasusuf/kube2consul)](https://goreportcard.com/report/github.com/hasusuf/kube2consul)
[![license](https://img.shields.io/github/license/hasusuf/kube2consul.svg?maxAge=2592000)](https://github.com/hasusuf/kube2consul/blob/master/LICENSE)

kube2consul connect to your Kubernetes cluster and mirror secrets that has name ended with `-secret` and configMaps that has name ended with `-config`
as well to Consul. At the moment this restriction is hardcoded. However it might be configurable in the future releases. 

### Prerequisites
* *Your `$KUBECONFIG` should be assigned to a valid `kubeconfig` file*.

## Installation
``` bash
$ RELEASE=$(curl --silent "https://api.github.com/repos/hasusuf/kube2consul/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
$ wget https://github.com/hasusuf/kube2consul/releases/download/$RELEASE/kube2consul-`uname -s`-`uname -m`
$ sudo install -m 755 kube2consul-`uname -s`-`uname -m` /usr/local/bin/kube2consul
```

### Usage
* Sync Kubernetes secrets/configMaps to Consul, outside the cluster <br />
`kube2consul sync --context minikube --namespace tools --consul-uri 10.83.46.158:8500`
* Sync Kubernetes secrets/configMaps to Consul, inside the cluster <br />
`kube2consul sync --namespace tools`
