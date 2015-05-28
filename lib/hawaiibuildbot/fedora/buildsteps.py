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

class ShellCommand(ShellMixin, steps.BuildStep):
    """
    Execute a shell command.
    """

    name = "sh"

    def __init__(self, **kwargs):
        steps.BuildStep.__init__(self, haltOnFailure=True, **kwargs)

    def getCurrentSummary(self):
        return {"step": u"running"}

    def getResultSummary(self):
        return {"step": u"success"}

    def _makeCommand(self, command):
        return self.makeRemoteShellCommand(collectStdout=True, collectStderr=True,
            command=command.split(" "))

    @defer.inlineCallbacks
    def _runCommand(self, command):
        cmd = yield self._makeCommand(command)
        yield self.runCommand(cmd)
        defer.returnValue(not cmd.didFail())

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
                self.setProperty("srpm", m.group(1), "BuildSourcePackage")
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
        # Compress upstream sources
        root = "fedora-{}-{}".format(self.distro, self.arch)
        resultdir = "../results"
        srpm = self.getProperty("srpm")
        command = "/usr/bin/mock --root={} --resultdir={} --rebuild {}".format(root, resultdir, srpm)
        cmd = yield self._makeCommand(command)
        yield self.runCommand(cmd)
        if cmd.didFail():
            defer.returnValue(FAILURE)
        defer.returnValue(SUCCESS)
