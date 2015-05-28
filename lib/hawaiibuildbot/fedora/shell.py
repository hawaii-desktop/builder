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
            command=command)

    @defer.inlineCallbacks
    def _runCommand(self, command):
        cmd = yield self._makeCommand(command)
        yield self.runCommand(cmd)
        defer.returnValue(not cmd.didFail())
