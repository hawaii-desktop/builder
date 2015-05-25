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

import os

from buildbot.plugins import steps
from buildbot.process.buildstep import ShellMixin, ShellMixin
from buildbot.status.results import *

from twisted.internet import defer

class PrepareChrootAction(ShellMixin, steps.BuildStep):
    """
    Create or update the chroot for the requested architecture.
    See https://wiki.archlinux.org/index.php/DeveloperWiki:Building_in_a_Clean_Chroot
    """

    def __init__(self, arch, **kwargs):
        steps.BuildStep.__init__(self, haltOnFailure=True, **kwargs)
        self.arch = arch
        self.name = "PrepareChroot %s" % self.arch
        self.cachedir = os.path.join(self.workdir, "..", "chroot")
        self.chrootdir = "root"

    @defer.inlineCallbacks
    def run(self):
        # Save the chroot base directory
        self.setProperty("chroot_basedir", self.cachedir, "PrepareChroot")
        # If the chroot directory is missing create the chroot
        path = os.path.join(self.cachedir, self.chrootdir)
        result = yield self._runCommand("test -d {}".format(path))
        if result:
            # Update the chroot
            self.workdir = self.cachedir
            result = yield self._runCommand("sudo arch-nspawn {} pacman -Syu --noconfirm".format(self.chrootdir))
            if result:
                defer.returnValue(SUCCESS)
            else:
                defer.returnValue(FAILURE)
        else:
            # Create the directory
            result = yield self._runCommand("mkdir -p {}".format(self.cachedir))
            if result:
                # Create the chroot
                self.workdir = self.cachedir
                result = yield self._runCommand("sudo mkarchroot {} base-devel".format(self.chrootdir))
                if result:
                    defer.returnValue(SUCCESS)
                else:
                    defer.returnValue(FAILURE)
            else:
                defer.returnValue(FAILURE)

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
