[![Build Status](https://travis-ci.org/xenserver/xe-guest-utilities.svg?branch=master)](https://travis-ci.org/xenserver/xe-guest-utilities)

# Introduction

This is the golang guest utilities for XenServer


XenStore golang client library
-----------
xe-guest-utilities.git/xenstoreclient


XenStore CLI
-----------
xe-guest-utilities.git/xenstore


Guest Utilities
-----------
xe-guest-utilities.git/xe-daemon


# Build Instructions

[Go development environment](https://golang.org/doc/install) is required to build the guest utilities.

After commit [94942cd597e](https://github.com/xenserver/xe-guest-utilities/commit/94942cd597ede2fb27a6b6a85ee6de364f19882c) guest utilities not support version <= 1.11, with modern go versions (later than 1.11) we can build with below guides:
## Build with GO111MODULE=off
In this case, project and source files are expected to put in GOPATH/src
1. Make sure go is installed in your environment, and set correctly in $PATH
2. Setup your go environment configurations

`GOROOT`
In newer versions, we don't need to set up the $GOROOT variable unless you use different Go versions

`GOPATH`
Go gets librarys from the directory `GOPATH`, so for the build to work, you need read/write permissions there. With `GO111MODULE` disabled, $GOPATH directory are expected to has below hierarchy. 
```bash
└── src
    ├── github.com
    │   └── xenserver
    │       └── xe-guest-utilities
    |
    └── golang.org
        └── x
            └── sys
```

`GO111MODULE`
Set `GO111MODULE` disabled
e.g.
let's say your project directory is /home/xe-guest-utilities-7.30.0
```bash
export GOPATH=/home/xe-guest-utilities-7.30.0
export GO111MODULE=off
```
3. Get the project

```bash
git clone https://github.com/xenserver/xe-guest-utilities.git $GOPATH/src/github.com/xenserver/xe-guest-utilities
```

4. Get external library

This project uses the `golang.org/x/sys/unix` library, you can use different methods to set the external library you use in your source code
```bash
go get -u golang.org/x/sys@latest
```
or
```bash
git clone git@github.com:golang/sys.git $GOPATH/src/golang.org/x/sys
```
5. Build
Go into the right directory `cd $GOPATH/src/github.com/xenserver/xe-guest-utilities`
now you can `make build` or `make`. Then you can get resulting files in `build/`, same layout as explained below
-----------
* The binarys will be in `build/obj`
* In `build/stage` are all required files and where they go when installed.
* In `build/dist` is a tarball with all files,symlinks and permissions.
-----------

## Build with GO111MODULE=on

In this case, we can place our project outside `$GOPATH`
1. Make sure go is installed in your environment, and set correctly in `$PATH`
2. Setup your go environment configurations

`GOPATH`
Go gets librarys from the `GOPATH`, so for this to work, you need read/write permissions there.If in doubt, set `GOPATH` to a temporary location, ie: `export GOPATH=$(pwd)` sets `GOPATH` to the local folder

`GO111MODULE`
With `GO111MODULE` enabled, go projects are no longer confined to $GOPATH, instead it use go.mod to keep track fo each package and it's version

e.g.
let's say your project directory is /home/xe-guest-utilities-7.30.0
```bash
# export GOPATH=/home/xe-guest-utilities-7.30.0
# export GO111MODULE=on
```
3. Get the project
```bash
git clone https://github.com/xenserver/xe-guest-utilities.git $GOPATH/xe-guest-utilities`
```

4. Set external library

This project uses the `golang.org/x/sys/unix` library, you can use different method to set the external library
* Download to `$GOPATH` manually
```bash
git clone git@github.com:golang/sys.git $GOPATH/golang.org/x/sys
```
And then add below content into `go.mod` before `require golang.org/x/sys v0.0.0-20210414055047-fe65e336abe0` to manually point to the correct place to get module from the specific place.
```bash
replace golang.org/x/sys v0.0.0-20210414055047-fe65e336abe0 => ../golang.org/x/sys
```
Then sync the vendor directory by `go mod vendor`

* Use go module tool to get
Sync the vendor directory by `go mod vendor`
In this process go will download `golang.org/x/sys` of the version `v0.0.0-20210414055047-fe65e336abe0` to the the vendor directory and refresh `vendor/modules.txt`

5. Build

Go into the right directory `cd $GOPATH/xe-guest-utilities/`, then you can use `make build` or `make`.
resulting files are in `build/`, same layout as explained above


# Collected information, by lifetime

## static

* from /var/cache/xe-linux-distribution
  * data/os_*
* compiled in
  * attr/PVAddons/Installed = 1
  * attr/PVAddons/MajorVersion
  * attr/PVAddons/MinorVersion
  * attr/PVAddons/MicroVersion
  * attr/PVAddons/BuildVersion
* runtime-dependant
  * control/feature-balloon = [01]

## changes on event (network config, hotplug, resume, migration...)

* from ifconfig/ip
  * attr/vif/$VIFID/ipv[46]/%d = $ADDR
  * xenserver/attr/net-sriov-vf/$VIFID/ipv[46]/%d = $ADDR
* from pvs, mount, /sys/block/, xenstore
  * data/volumes/%d/...
    * .../extents/0 = $BACKEND
    * .../name = /dev/xvd$X$N($PARTUUID) or /dev/xvd$X$N
    * .../size = $SIZE_IN_BYTES
    * .../mount_points/0 = $DIR or "[LVM]"
    * .../filesystem = $FSTYPE
* from /proc/meminfo
  * data/meminfo_total (or even static?)

## ephemeral
* from /proc/meminfo
  * data/meminfo_free
* from pvs or free
  * data/volumes/%d/free
* data/updated: date of last update