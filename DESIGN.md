# Builder design

## Basics

We need a high performance job scheduling system, specifically designed
to build Linux distribution packages and images.

It must use less resources as possible (CPU, RAM and bandwidth) so we
can run it on cheap hosting services.

Deployment must be easy.

## Actors

Builder has the following actors:

* master
* slave(s)
* command line interface
* web interface
* repository builder

The master collects the jobs, dispatches them to the appropriate slaves,
coordinates and has direct access to the database.

Slaves run on a variety of operating systems
Each of them subscribes to the master for certain channels, has support
for specific architectures and a maximum capacity that determines how
many jobs it can run in parallel.

After subscribing to the master, slaves pick up jobs and start processing.
A slave produces artifacts and those can either be packages or images.

The master is a service exposed to Internet, slaves instead are just
clients which means that only the master needs a dedicated server.

Packages produced by a slave are sent to the master which stores them
into an incoming area watched by the repository builder that recognize
which packages were built, deletes old versions, move the new ones
into the repository and rebuilds metadata.

Images produced by a slave are sent to the master and stored into
their directory. Once tested, a developer can decide to promote
one of them to be an official release.

Builds can be triggered either by a command line application, a web
interface that also shows progress and statistics or web hooks such
as those from GitHub.

Status information can also be sent to a web application like Slack or
IRC channels.

## Protocol

Builder uses a full duplex communication channel and a RPC protocol
between master and slaves with TLS support.

Slave certificates are signed by the master certification authority.

This protocol is used to send jobs to the slaves and exchange files
between them, that is artifacts produced by a slave or files fetched
from master.

The master also send job information to web clients through a web
socket.

## Package building

A couple of package types are supported: CI style packages and release
packages.

CI style means that packaging is as generic as possible and not for
a particular version, sources are fetched from a VCS.

Release packages instead are normal packages that target a particular
tarball released by upstream.

## CI style

* Update upstream and downstream git clones
* Create SRPM
* Query package database to know the previously built version
* Compare SRPM version with latest build: build only if SRPM is
  more recent, if the versions compare to be the same (this is the
  case for CI builds of subsequent commits the same day) SRPM wins
  and the package is built.
* Do a mock init if it's the first time
* Configure mock to point to our RPM repository
* Build with mock
* Detect artifacts and creates a listing file
* Upload artifacts and listing file to the incoming directory

## Release

* Update git clone
* Query package database and compare versions: build only if SRPM
  version is greater
* Do a mock init if it's the first time
* Configure mock to point to our RPM repository
* Build with mock
* Detect artifacts and creates a listing file
* Upload artifacts and listing file to the incoming directory
