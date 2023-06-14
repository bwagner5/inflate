# inflate

Inflate creates test deployments with various scheduling constraints. 

## Usage:

```
> inflate --help
Usage:
  inflate [command]

Available Commands:
  create      create an inflatable or maybe a few
  delete      delete an inflatable or maybe a few
  get         get an inflatable or maybe a few
  help        Help about any command

Flags:
  -f, --file string         YAML Config File
  -h, --help                help for inflate
  -k, --kubeconfig string   path to the kubeconfig file (default "/Users/wagnerbm/k8s/karpenter-dev/karpenter-dev")
  -n, --namespace string    k8s namespace (default "inflate")
  -o, --output string       Output mode: [short wide yaml] (default "short")
      --verbose             Verbose output
      --version             version

Use "inflate [command] --help" for more information about a command.
```

```
> inflate create --help
create an inflatable or maybe a few

Usage:
  inflate create [flags]

Flags:
      --capacity-type-spread   add a capacity-type topology spread constraint
      --dry-run                Dry-run prints the K8s manifests without applying
  -h, --help                   help for create
      --host-network           use host networking
      --hostname-spread        add a hostname topology spread constraint
  -i, --image string           Container image to use (default "public.ecr.aws/eks-distro/kubernetes/pause:3.7")
      --random-suffix          add a random suffix to the deployment name
  -z, --zonal-spread           add a zonal topology spread constraint

Global Flags:
  -f, --file string         YAML Config File
  -k, --kubeconfig string   path to the kubeconfig file (default "/Users/wagnerbm/k8s/karpenter-dev/karpenter-dev")
  -n, --namespace string    k8s namespace (default "inflate")
  -o, --output string       Output mode: [short wide yaml] (default "short")
      --verbose             Verbose output
      --version             version
```

## Installation:

```
brew install bwagner5/wagner/inflate
```

Packages, binaries, and archives are published for all major platforms (Mac amd64/arm64 & Linux amd64/arm64):

Debian / Ubuntu:

```
[[ `uname -m` == "aarch64" ]] && ARCH="arm64" || ARCH="amd64"
OS=`uname | tr '[:upper:]' '[:lower:]'`
wget https://github.com/bwagner5/inflate/releases/download/v0.0.1/inflate_0.0.1_${OS}_${ARCH}.deb
dpkg --install inflate_0.0.2_linux_amd64.deb
inflate --help
```

RedHat:

```
[[ `uname -m` == "aarch64" ]] && ARCH="arm64" || ARCH="amd64"
OS=`uname | tr '[:upper:]' '[:lower:]'`
rpm -i https://github.com/bwagner5/inflate/releases/download/v0.0.1/inflate_0.0.1_${OS}_${ARCH}.rpm
```

Download Binary Directly:

```
[[ `uname -m` == "aarch64" ]] && ARCH="arm64" || ARCH="amd64"
OS=`uname | tr '[:upper:]' '[:lower:]'`
wget -qO- https://github.com/bwagner5/inflate/releases/download/v0.0.1/inflate_0.0.1_${OS}_${ARCH}.tar.gz | tar xvz
chmod +x inflate
```

## Examples: 

```
> inflate create --zonal-spread
Created inflate/inflate

> inflate get
NAMESPACE	NAME
inflate  	inflate

> inflate create --random-suffix --hostname-spread --host-network -n my-ns
Created my-ns/inflate-9797840640

> inflate get
NAMESPACE	NAME
inflate  	inflate
my-ns    	inflate-9797840640

> inflate delete --all
Successfully Deleted Inflates
```