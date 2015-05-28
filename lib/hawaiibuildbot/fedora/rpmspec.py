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

from buildbot.status.results import *

from twisted.internet import defer

from shell import ShellCommand

class RpmSpec:
    """
    Read package information from a spec file.
    """

    def __init__(self, specfile=None, **kwargs):
        ShellCommand.__init__(self, **kwargs)
        self.specfile = specfile
        self._name = None
        self._version = None
        self._provides = None
        self._requires = None
        self._loaded = False

    def load(self):
        shellCommand = ShellCommand()

        cmd = yield shellCommand._makeCommand("rpmspec -q --srpm --qf '%{name}\\n' " + self.specfile)
        yield shellCommand.runCommand(cmd)
        if cmd.didFail():
            return
        self._name = cmd.stdio.strip()

        cmd = yield shellCommand._makeCommand("rpmspec -q --srpm --qf '%{version}\\n' " + self.specfile)
        yield shellCommand.runCommand(cmd)
        if cmd.didFail():
            return
        self._version = cmd.stdio.strip()

        cmd = yield shellCommand._makeCommand("rpmspec -q --provides " + self.specfile)
        yield shellCommand.runCommand(cmd)
        if cmd.didFail():
            return
        provides = []
        l = cmd.stdio.strip().split("\n")
        for k, v in l.split(" = "):
            provides.append({"name": k, "version": v})
        self._provides = provides

        cmd = yield shellCommand._makeCommand("rpmspec -q --requires " + self.specfile)
        yield shellCommand.runCommand(cmd)
        if cmd.didFail():
            return
        self._requires = cmd.stdio.strip().split("\n")

        self._loaded = True

    # Read-only properties
    loaded = property(lambda self: self._loaded)
    name = property(lambda self: self._name)
    version = property(lambda self: self._version)
    provides = property(lambda self: self._provides)
    requires = property(lambda self: self._requires)
