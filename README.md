# inflate

DESCRIPTION HERE

## Usage:


```
Put Usage here
Usage:
  inflate [command]
...
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

EXAMPLES HERE