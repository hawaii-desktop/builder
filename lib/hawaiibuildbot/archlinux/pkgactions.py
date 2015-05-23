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
from buildbot.process.buildstep import BuildStep
from buildbot.status.results import *

from twisted.internet import defer

class BinaryPackageBuild(ShellMixin, BuildStep):
    """
    Builds a package in a clean chroot.
    See https://wiki.archlinux.org/index.php/DeveloperWiki:Building_in_a_Clean_Chroot
    """
    description = "Build a package in a clean chroot"
    artifacts = []

    def __init__(self, name, arch, depends, provides, **kwargs):
        BuildStep.__init__(self, haltOnFailure=True, **kwargs)
        self.name = "binary_package/{}/{}".format(name, arch)
        self.pkgname = name
        self.arch = arch
        self.depends = depends
        self.provides = provides

    @defer.inlineCallbacks
    def run(self):
        log = yield self.addLog("logs")

        yield log.addStdout(u"Depends: {}\n".format(self.depends))
        yield log.addStdout(u"Provides: {}\n".format(self.provides))

        # Check whether we already built the latest version
        cmd = yield self._makeRemoteCommand("{}/helpers/pkgver -l {}/PKGBUILD".format(self.workdir, "."))
        defer.returnValue(FAILURE)

    def _makeRemoteCommand(self, cmd):
        args = cmd.split(" ")
        return self.makeRemoteShellCommand(collectStdout=True, collectStderr=True,
            command=args)
