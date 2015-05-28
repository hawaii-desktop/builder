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
from buildbot.status.results import *

from twisted.internet import defer

from shell import ShellCommand

class Mock(ShellCommand):
    """
    Executes a mock command.
    """

    name = "mock"

    haltOnFailure = True
    flunkOnFailure = True

    renderables = ["root", "resultdir"]

    root = None
    resultdir = None

    def __init__(self, root=None, resultdir=None, **kwargs):
        ShellCommand.__init__(self, **kwargs)

        if root:
            self.root = root
        if resultdir:
            self.resultdir = resultdir

        if not self.root:
            config.error("Please specify a mock root")

        self.command = ["/usr/bin/mock", "--root", self.root]
        if self.resultdir:
            self.command += ["--resultdir", self.resultdir]

    @defer.inlineCallbacks
    def run(self):
        cmd = yield self._makeCommand(self.command)
        yield self._runCommand(cmd)
        if cmd.didFail():
            defer.returnValue(FAILURE)
        defer.returnValue(SUCCESS)

class MockBuildSRPM(Mock):
    """
    Create a source RPM with mock.
    """

    name = "mockbuildsrpm"

    specfile = None
    sources = "."

    def __init__(self, specfile=None, sources=None, **kwargs):
        Mock.__init__(self, **kwargs)

        if specfile:
            self.specfile = specfile
        if not self.specfile:
            config.error("Please specify a spec file")
        if sources:
            self.sources = sources
        if not self.sources:
            config.error("Please specify a sources directory")

        self.command += ["--buildsrpm", "--spec", self.specfile, "--sources", self.sources]

class MockRebuild(Mock):
    """
    Rebuild a source RPM with mock.
    """

    name = "mockrebuild"

    srpm = None

    def __init__(self, srpm=None, **kwargs):
        Mock.__init__(self, **kwargs)

        if srpm:
            self.srpm = srpm
        if not self.srpm:
            config.error("Please specify a srpm")

        self.command += ["--rebuild", self.srpm]
