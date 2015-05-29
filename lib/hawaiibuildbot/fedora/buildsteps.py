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

from buildbot import config
from buildbot.plugins import steps
from buildbot.process.buildstep import ShellMixin
from buildbot.status.results import *
from buildbot.steps.package.rpm.mock import MockBuildSRPM, MockRebuild

from twisted.internet import defer

from shell import ShellCommand
from rpmspec import RpmSpec

class PrepareSources(ShellCommand):
    """
    Create a tarball of the upstream sources and create the spec file.
    """

    def __init__(self, pkgname, **kwargs):
        ShellCommand.__init__(self, **kwargs)
        self.pkgname = pkgname
        self.name = "prepare {}".format(pkgname)

    @defer.inlineCallbacks
    def run(self):
        # Compress upstream sources
        result = yield self._runCommand("../../helpers/mksrc {}".format(self.pkgname))
        if result:
            defer.returnValue(SUCCESS)
        else:
            defer.returnValue(FAILURE)

class BuildSourcePackage(MockBuildSRPM):
    """
    Create a SRPM from sources using mock.
    """

    def __init__(self, arch, distro, pkgname, **kwargs):
        self.arch = arch
        self.distro = distro
        self.pkgname = pkgname
        root = "fedora-{}-{}".format(self.distro, self.arch)
        resultdir = "../results"
        spec = "{}.spec".format(self.pkgname)
        MockBuildSRPM.__init__(self, root=root, resultdir=resultdir, spec=spec, **kwargs)
        self.name = "srpm {} {}/{}".format(self.pkgname, self.distro, self.arch)

class AcquirePackageInfo(ShellCommand):
    """
    Retrieve information from a source package.
    """

    name = "packageinfo"

    pkgname = None
    specfile = None

    def __init__(self, pkgname=None, **kwargs):
        ShellCommand.__init__(self, **kwargs)

        if pkgname:
            self.pkgname = pkgname
        if not self.pkgname:
            config.error("You must specify a package name")

        self.specfile = "{}.spec".format(self.pkgname)
        self.name = "packageinfo {}".format(self.pkgname)

    @defer.inlineCallbacks
    def run(self):
        log = yield self.addLog("logs")

        # Find srpm
        cmd = yield self._makeCommand(["/usr/bin/find", "../results", "-type", "f", "-name", "*.src.rpm", "-printf", "%f "])
        yield self.runCommand(cmd)
        if cmd.didFail():
            yield log.addStderr(u"Unable to find srpm path\n")
            defer.returnValue(FAILURE)
        srpm = cmd.stdout.strip()
        yield log.addStdout(u"Source package: {}\n".format(srpm))

        # Retrieve package information
        rpmSpec = RpmSpec(specfile=self.specfile, workdir=self.workdir)
        rpmSpec.load()
        if not rpmSpec.loaded:
            yield log.addStderr(u"Unable to read specfile {}\n".format(self.specfile))
            defer.returnValue(FAILURE)
        # Save srpm location and package information
        pkg_info = self.getProperty("package_info") or {}
        pkg_info[self.pkgname] = {
            "name": rpmSpec.pkg_name,
            "version": rpmSpec.pkg_version,
            "provides": rpmInfo.pkg_provides,
            "requires": rpmInfo.pkg_requires,
            "srpm": srpm
        }
        yield log.addStdout(u"Package information for {}: {}\n".format(self.pkgname, pkg_info[self.pkgname]))
        self.setProperty("package_info", pkg_info, "BuildSourcePackage")
        defer.returnValue(SUCCESS)

class BuildBinaryPackage(MockRebuild):
    """
    Build a SRPM using mock.
    """

    def __init__(self, arch, distro, pkgname, **kwargs):
        self.arch = arch
        self.distro = distro
        self.pkgname = pkgname
        root = "fedora-{}-{}".format(self.distro, self.arch)
        resultdir = "../results"
        pkg_info = self.getProperty("package_info")
        if len(pkg_info.get(self.pkgname, {}).keys()) == 0:
            config.error("Unable to retrieve package information")
        srpm = package_info[self.pkgname]["srpm"]
        MockRebuild.__init__(self, root=root, resultdir=resultdir, srpm=srpm, **kwargs)
        self.name = "rpm {} {}/{}".format(self.pkgname, self.distro, self.arch)
