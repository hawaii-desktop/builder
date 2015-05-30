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

# Portions Copyright Buildbot Team Members
# Portions Copyright Marius Rieder <marius.rieder@durchmesser.ch>

import os, re

from buildbot import config
from buildbot.process import remotecommand, logobserver
from buildbot.status.results import *
from buildbot.steps.shell import ShellCommand

from twisted.internet import defer

#from shell import ShellCommand

class MockStateObserver(logobserver.LogLineObserver):
    _line_re = re.compile(r'^.*State Changed: (.*)$')

    def outLineReceived(self, line):
        m = self._line_re.search(line.strip())
        if m:
            state = m.group(1)
            if not state == "end":
                self.step.descriptionSuffix = ["[%s]" % m.group(1)]
            else:
                self.step.descriptionSuffix = None
            self.step.step_status.setText(self.step.describe(False))

class Mock(ShellCommand):
    """
    Executes a mock command.
    """

    name = "mock"

    haltOnFailure = True
    flunkOnFailure = True

    renderables = ["root", "resultdir"]

    mock_logfiles = ["build.log", "root.log", "state.log"]

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

    def start(self):
        # Observe mock logs
        if self.resultdir:
            for lname in self.mock_logfiles:
                self.logfiles[lname] = self.build.path_module.join(self.resultdir,
                                                                   lname)
        else:
            for lname in self.mock_logfiles:
                self.logfiles[lname] = lname
        self.addLogObserver("state.log", MockStateObserver())

        # Remove old logs
        cmd = remotecommand.RemoteCommand("rmdir", {"dir":
                                                    map(lambda l: self.build.path_module.join("build", self.logfiles[l]),
                                                        self.mock_logfiles)})
        d = self.runCommand(cmd)

        @d.addCallback
        def removeDone(cmd):
            ShellCommand.start(self)
        d.addErrback(self.failed)

class MockBuildSRPM(Mock):
    """
    Create a source RPM with mock.
    """

    name = "mockbuildsrpm"

    description = ["mock buildsrpm"]
    descriptionDone = ["mock buildsrpm"]

    spec = None
    sources = "."

    def __init__(self, spec=None, sources=None, **kwargs):
        Mock.__init__(self, **kwargs)

        if spec:
            self.spec = spec
        if not self.spec:
            config.error("Please specify a spec file")
        if sources:
            self.sources = sources
        if not self.sources:
            config.error("Please specify a sources directory")

        self.command += ["--buildsrpm", "--spec", self.spec, "--sources", self.sources]

        self.addLogObserver("stdio",
            logobserver.LineConsumerLogObserver(self.logConsumer))

    # Read-only properties
    srpm = property(lambda self: self._srpm)

    def logConsumer(self):
        r = re.compile(r"Wrote: .*/([^/]*.src.rpm)")
        while True:
            stream, line = yield
            m = r.search(line)
            if m:
                self.setProperty("srpm", _self.build.path_module.join(self.resultdir, m.group(1)), "MockBuildSRPM")

class MockRebuild(Mock):
    """
    Rebuild a source RPM with mock.
    """

    name = "mockrebuild"

    description = ["mock rebuilding srpm"]
    descriptionDone = ["mock rebuild srpm"]

    srpm = None

    def __init__(self, srpm=None, **kwargs):
        Mock.__init__(self, **kwargs)

        if srpm:
            self.srpm = srpm
        if not self.srpm:
            config.error("Please specify a srpm")

        self.command += ["--rebuild", self.srpm]
