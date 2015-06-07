#
# This file is part of Hawaii.
#
# Copyright (C) 2015 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
#
# Author(s):
#    Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
#
# $BEGIN_LICENSE:GPL3$
#
# This program is free software; you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation; version 3 (with exceptions) or any
# later version accepted by Pier Luigi Fiorini, which shall act as a
# proxy defined in Section 14 of version 3 of the license.
#
# Exceptions are described in Hawaii GPL Exception version 1.0,
# included in the file GPL3_EXCEPTION.txt in this package.
#
# Any modifications to this file must keep this entire header intact.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.
#
# $END_LICENSE$
#

import os

from buildbot.plugins import steps
from buildbot.process.buildstep import ShellMixin, ShellMixin
from buildbot.status.results import *

from twisted.internet import defer

class PrepareChroot(ShellMixin, steps.BuildStep):
    """
    Create or update the chroot for the requested architecture.
    See https://wiki.archlinux.org/index.php/DeveloperWiki:Building_in_a_Clean_Chroot
    BEWARE: This only support the same architecture of the slave.
    """

    def __init__(self, arch, **kwargs):
        steps.BuildStep.__init__(self, haltOnFailure=True, **kwargs)
        self.arch = arch
        self.name = "chroot %s" % self.arch
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

class CcmAction(ShellMixin, steps.BuildStep):
    """
    Build packages and manages chroots with clean-chroot-manager.
    See https://bbs.archlinux.org/viewtopic.php?id=168421
    """

    def __init__(self, arch, action, **kwargs):
        steps.BuildStep.__init__(self, haltOnFailure=True, **kwargs)
        self.arch = arch
        action_map = {"c": "Create", "u": "Update", "s": "Build"}
        self.name = "ccm{} {}".format(action_map[action], self.arch)
        self.action = action

    @defer.inlineCallbacks
    def run(self):
        cmd = yield self._makeShellCommand(["../helpers/ccm-setup", "../chroot"])
        yield self.runCommand(cmd)
        if cmd.didFail():
            defer.returnValue(FAILURE)

        cmd = yield self._makeCcmCommand(self.action)
        yield self.runCommand(cmd)
        if cmd.didFail():
            defer.returnValue(FAILURE)
        else:
            defer.returnValue(SUCCESS)

    def getCurrentSummary(self):
        return {"step": u"running"}

    def getResultSummary(self):
        return {"step": u"success"}

    def _makeCcmCommand(self, action, **kwargs):
        bits = "32" if self.arch == "i686" else "64"
        return self.makeRemoteShellCommand(collectStdout=True, collectStderr=True,
            command=["sudo", "linux" + bits, "ccm" + bits, action], **kwargs)

    def _makeShellCommand(self, args, **kwargs):
        return self.makeRemoteShellCommand(collectStdout=True, collectStderr=True,
            command=args, **kwargs)

class PrepareCcm(CcmAction):
    """
    Create or update a chroot with clean-chroot-manager.
    See https://bbs.archlinux.org/viewtopic.php?id=168421
    """

    def __init__(self, arch, **kwargs):
        CcmAction.__init__(self, arch, "c", **kwargs)
        self.name = "prepare-ccm {}".format(self.arch)

    @defer.inlineCallbacks
    def run(self):
        cmd = yield self._makeShellCommand(["../helpers/ccm-setup", "../chroot"])
        yield self.runCommand(cmd)
        if cmd.didFail():
            defer.returnValue(FAILURE)

        # If the chroot directory is missing create the chroot
        bits = "32" if self.arch == "i686" else "64"
        result = yield self._runCommand("test -d ../chroot{}/root".format(bits))
        if result:
            action = "u"
        else:
            action = "c"

        cmd = yield self._makeCcmCommand(action)
        yield self.runCommand(cmd)
        if cmd.didFail():
            defer.returnValue(FAILURE)
        else:
            defer.returnValue(SUCCESS)

    def _makeCommand(self, command):
        return self.makeRemoteShellCommand(collectStdout=True, collectStderr=True,
            command=command.split(" "))

    @defer.inlineCallbacks
    def _runCommand(self, command):
        cmd = yield self._makeCommand(command)
        yield self.runCommand(cmd)
        defer.returnValue(not cmd.didFail())
