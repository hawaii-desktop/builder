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

from buildbot.plugins import steps
from buildbot.process.buildstep import ShellMixin
from buildbot.status.results import *

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

class BuildSourcePackage(ShellCommand):
    """
    Create a SRPM from sources using mock.
    """

    def __init__(self, arch, distro, pkgname, **kwargs):
        ShellCommand.__init__(self, **kwargs)
        self.arch = arch
        self.distro = distro
        self.pkgname = pkgname
        self.name = "srpm {} {}/{}".format(self.pkgname, self.distro, self.arch)

    @defer.inlineCallbacks
    def run(self):
        log = yield self.addLog("logs")

        # Compress upstream sources
        root = "fedora-{}-{}".format(self.distro, self.arch)
        spec = "{}.spec".format(self.pkgname)
        sources = "."
        resultdir = "../results"
        command = "/usr/bin/mock --root={} --resultdir={} --buildsrpm --spec {} --sources {}".format(root, resultdir, spec, sources)
        cmd = yield self._makeCommand(command)
        yield self.runCommand(cmd)
        if cmd.didFail():
            defer.returnValue(FAILURE)
        else:
            import re
            r = re.compile(r"Wrote: .*/([^/]*.src.rpm)")
            m = r.search(cmd.stdout.strip())
            if m:
                # Retrieve package information
                rpmSpec = RpmSpec(specfile=spec)
                rpmSpec.load()
                if not rpmSpec.loaded:
                    yield log.addStderr(u"Unable to read specfile")
                    defer.returnValue(FAILURE)
                # Save srpm location and package information
                pkg_info = self.getProperty("package_info") or {}
                pkg_info[self.pkgname] = {
                    "name": rpmSpec.pkg_name,
                    "version": rpmSpec.pkg_version,
                    "provides": rpmInfo.pkg_provides,
                    "requires": rpmInfo.pkg_requires,
                    "srpm": m.group(1)
                }
                yield log.addStdout(u"Package information for {}: {}\n".format(self.pkgname, pkg_info[self.pkgname]))
                self.setProperty("package_info", pkg_info, "BuildSourcePackage")
            else:
                defer.returnValue(FAILURE)
        defer.returnValue(SUCCESS)

class BuildBinaryPackage(ShellCommand):
    """
    Build a SRPM using mock.
    """

    def __init__(self, arch, distro, pkgname, **kwargs):
        ShellCommand.__init__(self, **kwargs)
        self.arch = arch
        self.distro = distro
        self.pkgname = pkgname
        self.name = "rpm {} {}/{}".format(self.pkgname, self.distro, self.arch)

    @defer.inlineCallbacks
    def run(self):
        log = yield self.addLog("logs")

        # Compress upstream sources
        root = "fedora-{}-{}".format(self.distro, self.arch)
        resultdir = "../results"
        pkg_info = self.getProperty("package_info")
        if len(pkg_info.get(self.pkgname, {}).keys()) == 0:
            yield log.addStderr(u"Unable to retrieve package information")
            defer.returnValue(FAILURE)
        srpm = package_info[self.pkgname]["srpm"]
        command = "/usr/bin/mock --root={} --resultdir={} --rebuild {}".format(root, resultdir, srpm)
        cmd = yield self._makeCommand(command)
        yield self.runCommand(cmd)
        if cmd.didFail():
            defer.returnValue(FAILURE)
        defer.returnValue(SUCCESS)
