#
# This file is part of Hawaii.
#
# Copyright (C) 2015 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 2 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.
#

import os, re

from buildbot.steps.shell import ShellCommand
from buildbot.plugins import steps
from buildbot.process.buildstep import ShellMixin
from buildbot.status.results import *

from twisted.internet import defer

from mock import MockBuildSRPM

class PrepareSources(ShellCommand):
    """
    Create a tarball of the upstream sources and create the spec file.
    """

    def __init__(self, pkgname, **kwargs):
        ShellCommand.__init__(self, **kwargs)
        self.command = ["../../helpers/mksrc", pkgname]
        self.name = "{} prepare".format(pkgname)

class SourcePackage(MockBuildSRPM):
    """
    Create a SRPM from sources using mock.
    """

    def __init__(self, arch, distro, pkgname, **kwargs):
        root = "fedora-{}-{}".format(distro, arch)
        resultdir = "../srpm"
        spec = "{}.spec".format(pkgname)
        MockBuildSRPM.__init__(self, root=root, resultdir=resultdir, spec=spec, **kwargs)

        self.arch = arch
        self.distro = distro
        self.pkgname = pkgname
        self.name = "{} srpm {}".format(self.pkgname, root)

class BuildSourcePackages(ShellMixin, steps.BuildStep):
    """
    Move SRPMs to a common location and chain build them.
    """

    name = "chain"

    def __init__(self, pkgnames, arch, distro, **kwargs):
        kwargs = self.setupShellMixin(kwargs, prohibitArgs=["command"])
        steps.BuildStep.__init__(self, haltOnFailure=True, **kwargs)
        self.pkgnames = pkgnames
        self.arch = arch
        self.distro = distro

    @defer.inlineCallbacks
    def run(self):
        log = yield self.addLog("logs")

        # Make a list with package information
        pkg_info = []
        for pkgname in self.pkgnames:
            # Spec file path
            path = "{}/downstream/{}.spec".format(pkgname, pkgname)
            # Retrieve NVR
            cmd = yield self._makeCommand(["../helpers/spec-nvr", path])
            yield self.runCommand(cmd)
            if cmd.didFail():
                yield log.addStderr(u"Unable to determine NVR from \"{}\"\n".format(path))
                defer.returnValue(FAILURE)
            n, e, v, r = cmd.stdout.strip().split(" ")
            # Populate list
            pkg_info.append({"name": n, "epoch": e, "version": v, "release": r})
        self.setProperty("pkg_info", pkg_info, "MoveSourcePackages")
        yield log.addStdout(u"Package information: {}\n".format(pkg_info))

        # Find SRPMs
        srpms = []
        for pkg in pkg_info:
            # Find the artifacts
            path = "{}/srpm".format(pkgname)
            cmd = yield self._makeCommand(["/usr/bin/find", path, "-type", "f", "-name", "*.src.rpm"])
            yield self.runCommand(cmd)
            if cmd.didFail():
                defer.returnValue(FAILURE)

            # Add artifact to the list
            artifacts = cmd.stdout.strip().split(" ")
            yield log.addStdout(u"Artifacts for {}: {}\n".format(pkg["name"], artifacts))
            if pkg["epoch"] == "(none)":
                r = re.compile(r'.*\-{}\-{}\.src\.rpm'.format(pkg["version"], pkg["release"]))
            else:
                r = re.compile(r'.*\-{}:{}\-{}\.src\.rpm'.format(pkg["epoch"], pkg["version"], pkg["release"]))
            matching_artifacts = filter(r.match, artifacts)
            yield log.addStdout(u"Matching artifacts for {}: {}\n".format(pkg["name"], matching_artifacts))
            if len(matching_artifacts) == 0:
                yield log.addStderr(u"No matching artifacts found for {}\n".format(pkg["name"]))
                defer.returnValue(FAILURE)
            srpms += matching_artifacts
        yield log.addStdout(u"SRPMs: {}\n".format(srpms))

        # Move all SRPMs
        for srpm in srpms:
            cmd = yield self._makeCommand(["mv", "-f", srpm, "srpms"])
            yield self.runCommand(cmd)
            if cmd.didFail():
                defer.returnValue(FAILURE)

        # Chain build
        srpms = [os.path.basename(x) for x in srpms]
        root = "fedora-{}-{}".format(self.distro, self.arch)
        step = ChainBuild(root=root, srpms=srpms, workdir="srpms")
        self.build.addStepsAfterCurrentStep([step])

        defer.returnValue(SUCCESS)

    def _makeCommand(self, args, **kwargs):
        return self.makeRemoteShellCommand(collectStdout=True, collectStderr=True,
            command=args, **kwargs)

    @defer.inlineCallbacks
    def _runCommand(self, command, **kwargs):
        cmd = yield self._makeCommand(command, **kwargs)
        yield self.runCommand(cmd)
        defer.returnValue(not cmd.didFail())

class ChainBuild(ShellCommand):
    """
    Build RPMS using mockchain.
    """

    def __init__(self, root, srpms, **kwargs):
        ShellCommand.__init__(self, **kwargs)
        self.name = "mockchain {}".format(root)
        self.command = ["/usr/bin/mockchain", "--root", root, "--tmp_prefix=buildbot", " ".join(srpms)]
