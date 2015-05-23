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

from buildbot.process.buildstep import ShellMixin
from buildbot.process.buildstep import BuildStep
from buildbot.status.results import *

from twisted.internet import defer

class BaseChrootAction(ShellMixin, BuildStep):
    """
    Base chroot action.
    See https://wiki.archlinux.org/index.php/DeveloperWiki:Building_in_a_Clean_Chroot
    """

    def __init__(self, arch, **kwargs):
        BuildStep.__init__(self, haltOnFailure=True, **kwargs)
        self.arch = arch

    @defer.inlineCallbacks
    def run(self):
        cmd = yield self._makeCommand()
        yield self.runCommand(cmd)
        if cmd.didFail():
            defer.returnValue(FAILURE)
        else:
            defer.returnValue(SUCCESS)

    def getCurrentSummary(self):
        return {"step": u"running"}

    def getResultSummary(self):
        return {"step": u"success"}

class CreateChrootAction(BaseChrootAction):
    """
    Create a chroot to build packages.
    See https://wiki.archlinux.org/index.php/DeveloperWiki:Building_in_a_Clean_Chroot
    """

    def __init__(self, arch, **kwargs):
        BaseChrootAction.__init__(self, arch, **kwargs)
        self.name = u"create-chroot %s" % self.arch

    def _makeCommand(self):
        return self.makeRemoteShellCommand(collectStdout=True, collectStderr=True,
            command=["sudo", "mkarchroot", "chroot/root", "base-devel"])

class UpdateChrootAction(BaseChrootAction):
    """
    Update a chroot to build packages.
    See https://wiki.archlinux.org/index.php/DeveloperWiki:Building_in_a_Clean_Chroot
    """

    def __init__(self, arch, **kwargs):
        BaseChrootAction.__init__(self, arch, **kwargs)
        self.name = u"update-chroot %s" % self.arch

    def _makeCommand(self):
        return self.makeRemoteShellCommand(collectStdout=True, collectStderr=True,
            command=["sudo", "arch-nspawn", "chroot/root", "pacman", "-Syu"])

class CreateOrUpdateChrootAction(BaseChrootAction):
    """
    Create or update a chroot to build packages.
    See https://wiki.archlinux.org/index.php/DeveloperWiki:Building_in_a_Clean_Chroot
    """

    def __init__(self, arch, **kwargs):
        BaseChrootAction.__init__(self, arch, **kwargs)
        self.name = u"create-or-update-chroot %s" % self.arch

    @defer.inlineCallbacks
    def run(self):
        cmd = yield self._makeRemoteCommand("ls chroot/root")
        yield self.runCommand(cmd)
        if cmd.didFail():
            cmd = yield self._makeRemoteCommand("sudo mkarchroot chroot/root base-devel")
            yield self.runCommand(cmd)
            if cmd.didFail():
                defer.returnValue(FAILURE)
        else:
            cmd = yield self._makeRemoteCommand("sudo arch-nspawn chroot/root pacman -Syu")
            yield self.runCommand(cmd)
            if cmd.didFail():
                defer.returnValue(FAILURE)
        defer.returnValue(SUCCESS)

    def _makeRemoteCommand(self, cmd):
        return self.makeRemoteShellCommand(collectStdout=True, collectStderr=True,
            command=cmd.split(" "))
