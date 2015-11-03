# Hacking Guide

## Overview

This guide contains instructions for those looking to hack on rkt.
For more information on the rkt internals, see the [`devel`](devel/) documentation.

## Building rkt

rkt should be able to be built on any modern Linux system.
For the most part the codebase is self-contained (e.g. all dependencies are vendored), but assembly of the stage1 requires some other tools to be installed on the system.

### Build-time requirements

* Linux 3.8+
  * make
  * gcc
  * glibc development and static pieces (on Fedora/RHEL/Centos: glibc-devel and glibc-static packages, on Debian/Ubuntu libc6-dev package)
  * cpio
  * squashfs-tools
  * realpath
  * gpg
  * autoconf
* Go 1.4+

Once the requirements have been met you can build rkt by running the following commands:

```
git clone https://github.com/coreos/rkt.git
cd rkt
./autogen.sh && ./configure && make
```

Build verbosity can be controlled with the V variable.
Set V to 0 to have a silent build.
Set V to either 1 or 2 to get short messages about what is being done (level 2 prints more of them).
Set V to 3 to get raw output.
Instead of a number, english words can be used.
`quiet` or `silent` for level 0, `info` for level 1, `all` for level 2 and `raw` for level 3. Example:

`make V=raw`

### Run-time requirements

rkt is statically linked and does not require any dynamic libraries to be installed. However, it requires the following kernel features:

* `CONFIG_CGROUPS`
* `CONFIG_NAMESPACES`
* `CONFIG_UTS_NS`
* `CONFIG_IPC_NS`
* `CONFIG_PID_NS`
* `CONFIG_NET_NS`

Additionally, the following features are nice to have:

* `CONFIG_OVERLAY_FS` (to prepare the rootfs without tar)

rkt needs the following bug fixes in Linux:
* [ovl: fix open in stacked overlay](https://github.com/torvalds/linux/commit/1c8a47df36d72ace8cf78eb6c228aa0f8027d3c2) (fix merged in Linux 4.3)
* [cpuset: use trialcs->mems_allowed as a temp variable](https://github.com/torvalds/linux/commit/24ee3cf89bef04e8bc23788aca4e029a3f0f06d9) (fix merged in Linux 4.2)

### With Docker

Alternatively, you can build rkt in a Docker container with the following command.
Replace $SRC with the absolute path to your rkt source code:

```
# docker run -v $SRC:/opt/rkt -i -t golang:1.4 /bin/bash -c "apt-get update && apt-get install -y coreutils cpio squashfs-tools realpath autoconf file && cd /opt/rkt && go get github.com/appc/spec/... && ./autogen.sh && ./configure && make"
```

### Building systemd in stage1 from the sources

By default, rkt gets systemd from a CoreOS image to generate stage1. But it's also possible to build systemd from the sources.
After running `./autogen.sh` you can select the following options:

* `./configure --with-stage1-flavors=coreos,src,host,kvm`: choose which flavors to build (default: 'coreos,kvm') (kvm also uses systemd from CoreOS, the difference is in the execution engine, see next section)
* `./configure --with-stage1-default-flavor=coreos|src|host|kvm`: choose which built flavor should be the default (default: first from the flavors list)
* `./configure --with-stage1-systemd-version=version`: if 'src' flavor is built, choose the systemd branch or tag to build (default: 'v222')
* `./configure --with-stage1-systemd-src=git-path`: if 'src' flavor is built, systemd git repository's address (default: 'https://github.com/systemd/systemd.git')

Example:

```
./autogen.sh && ./configure --with-stage1-flavors=src --with-stage1-systemd-version=master --with-stage1-systemd-src=$HOME/src/systemd && make
```

### Building stage1 with kvm as execution engine

The stage1 kvm image is based on CoreOS, but with additional components for running containers on top of a hypervisor.

To build, use `--with-stage1-flavors=kvm` flag in `./configure`

This will generate stage1 with embedded kernel and kvmtool to start pod in virtual machine.

Additional build dependencies for the stage1 kvm follow. If building with docker, these must be added to the `apt-get install` command.

* wget
* xz-utils
* patch
* bc

### Alternative stage1 paths

rkt is designed and intended to be modular, using a [staged architecture](devel/architecture.md).

`rkt run` determines the stage1 image it should use via its `-stage1-image` flag.
By default, if this flag is unset at runtime, rkt will default to the configure-time settings. It usually means that rkt will look for a file called `stage1-<default flavor>.aci` that is in the same directory as the rkt binary itself.

However, a default value can be set for this parameter at build time by setting the option `--with-stage1-default-location` when invoking `./configure`
This is useful for those packaging rkt for distribution who will provide the stage1 in a fixed/known location.

The option should be set to the fully qualified path at which rkt can find the stage1 image - for example:

```
./autogen.sh && ./configure --with-stage1-default-location=/usr/lib/rkt/stage1.aci && make
```

rkt will then use this environment variable to set the default value for the `stage1-image` flag.


## Managing Dependencies

rkt uses [`godep`](https://github.com/tools/godep) to manage third-party dependencies.
The build process is crafted to make this transparent to most users (i.e. if you're just building rkt from source, or modifying any of the codebase without changing dependencies, you should have no need to interact with godep).
But occasionally the need arises to either a) add a new dependency or b) update/remove an existing dependency.
At this point, the ramblings below from an experienced Godep victim^Wenthusiast might prove of use...

### Update godep

Step zero is generally to ensure you have the **latest version** of `godep` available in your `PATH`.

### Having the right directory layout (i.e. `GOPATH`)

To work with `godep`, you'll need to have the repository (i.e. `github.com/coreos/rkt`) checked out in a valid `GOPATH`.
If you use the [standard Go workflow](https://golang.org/doc/code.html#Organization), with every package in its proper place in a workspace, this should be no problem.
As an example, if one was obtaining the repository for the first time, one would do the following:

```
$ export GOPATH=/tmp/foo               # or any directory you please
$ go get -d github.com/coreos/rkt/...  # or 'git clone https://github.com/coreos/rkt $GOPATH/src/github.com/coreos/rkt'
$ cd $GOPATH/src/github.com/coreos/rkt
```

If, however, you instead prefer to manage your source code in directories like `~/src/rkt`, there's a problem: `godep` doesn't like symbolic links (which is what the rkt build process uses by default to create a self-contained GOPATH).
Hence, you'll need to work around this with bind mounts, with something like the following:

```
$ export GOPATH=/tmp/foo        # or any directory you please
$ mkdir -p $GOPATH/src/github.com/coreos/rkt
# mount --bind ~/src/rkt $GOPATH/src/github.com/coreos/rkt
$ cd $GOPATH/src/github.com/coreos/rkt
```

One benefit of this approach over the single-workspace workflow is that checking out different versions of dependencies in the `GOPATH` (as we are about to do) is guarnteed to not affect any other packages in the `GOPATH`.
(Using [gvm](https://github.com/moovweb/gvm) or other such tomfoolery to manage `GOPATH`s is an exercise left for the reader.)

### Restoring the current state of dependencies

Now that we have a functional `GOPATH`, use `godep` to restore the full set of vendored dependencies to their correct versions.
(What this command does is essentially just loop over the set of dependencies codified in `Godeps/Godeps.json`, using `go get` to retrieve and then `git checkout` (or equivalent) to set each to their correct revision.)

```
$ godep restore # might take a while if it's the first time...
```

At this stage, your path forks, depending on what exactly you want to do: add, update or remove a dependency.
But in _all three cases_, the procedure finishes with the [same save command](#saving-the-set-of-dependencies).

#### Add a new dependency

In this case you'll first need to retrieve the dependency you're working with into `GOPATH`.
As a simple example, assuming we're adding `github.com/fizz/buzz`:

```
$ go get -d github.com/fizz/buzz
```

Then add your new dependency into `godep`'s purview by simply importing the standard package name in one of your sources:

```
$ vim $GOPATH/src/github.com/coreos/rkt/some/file.go
...
import "github.com/fizz/buzz"
...
```

Now, GOTO [saving](#saving-the-set-of-dependencies)

#### Update an existing dependency

In this case, assuming we're updating `github.com/foo/bar`:

```
$ cd $GOPATH/src/github.com/foo/bar
$ git pull   # or 'go get -d -u github.com/foo/bar/...'
$ git checkout $DESIRED_REVISION
$ cd $GOPATH/src/github.com/coreos/rkt
$ godep update github.com/foo/bar/...
```

Now, GOTO [saving](#saving-the-set-of-dependencies)

#### Removing an existing dependency

This is the simplest case of all: simply remove all references to a dependency from the source files.

Now, GOTO [saving](#saving-the-set-of-dependencies)

### Saving the set of dependencies

Finally, here we are, the magic command, the holy grail, the ultimate conclusion of all `godep` operations.
Provided you have followed the preceding instructions, regardless of whether you are adding/removing/modifying dependencies, this command will cast the necessary spells to solve all of your dependency worries:

```
$ godep save -r ./...
```

## Finishing up

At this point, you should be good to PR.
As well as a simple sanity check that the code actually builds and tests pass, here are some things to look out for:
- `git status Godeps/` should show only a minimal and relevant change (i.e. only the dependencies you actually intended to touch).
- `git diff Godeps/` should be free of any changes to import paths within the vendored dependencies
- `git diff` should show _all_ third-party import paths prefixed with `Godeps/_workspace`

If something looks awry, restart, pray to your preferred deity, and try again.
