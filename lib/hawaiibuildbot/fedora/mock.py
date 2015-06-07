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

class MockChainStateObserver(logobserver.LogLineObserver):
    _line_re = re.compile(r'^.*(Start|Finish): (.*)$')

    def outLineReceived(self, line):
        m = self._line_re.search(line.strip())
        if m:
            what = m.group(1)
            state = m.group(2)
            if what == "Start":
                self.step.descriptionSuffix = ["[%s]" % state]
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

    def getConfiguration(self, ccache=True, yum_cache=True, root_cache=True,
                         tmpfs=True, nosync=False, sign=True, gpg_name=None, gpg_path=None):
        config = """
config_opts['nosync'] = """ + str(nosync) + """
config_opts['plugin_conf']['package_state_enable'] = False
config_opts['plugin_conf']['ccache_enable'] = """ + str(ccache) + """
config_opts['plugin_conf']['ccache_opts'] = {}
config_opts['plugin_conf']['ccache_opts']['max_cache_size'] = '4G'
config_opts['plugin_conf']['ccache_opts']['compress'] = None
config_opts['plugin_conf']['ccache_opts']['dir'] = "%(cache_topdir)s/%(root)s/ccache/u%(chrootuid)s/"
config_opts['plugin_conf']['yum_cache_enable'] = """ + str(yum_cache) + """
config_opts['plugin_conf']['yum_cache_opts'] = {}
config_opts['plugin_conf']['yum_cache_opts']['max_age_days'] = 30
config_opts['plugin_conf']['yum_cache_opts']['max_metadata_age_days'] = 30
config_opts['plugin_conf']['yum_cache_opts']['dir'] = "%(cache_topdir)s/%(root)s/%(package_manager)s_cache/"
config_opts['plugin_conf']['yum_cache_opts']['target_dir'] "/var/cache/%(package_manager)s/"
config_opts['plugin_conf']['yum_cache_opts']['online'] = True
config_opts['plugin_conf']['root_cache_enable'] = """ + str(root_cache) + """
config_opts['plugin_conf']['root_cache_opts'] = {}
config_opts['plugin_conf']['root_cache_opts']['age_check'] = True
config_opts['plugin_conf']['root_cache_opts']['max_age_days'] = 15
config_opts['plugin_conf']['root_cache_opts']['dir'] = "%(cache_topdir)s/%(root)s/root_cache/"
config_opts['plugin_conf']['root_cache_opts']['compress_program'] = "pigz"
config_opts['plugin_conf']['root_cache_opts']['extension'] = ".gz"
config_opts['plugin_conf']['root_cache_opts']['exclude_dirs'] = ["./proc", "./sys", "./dev",
                                                                 "./tmp/ccache", "./var/cache/yum" ]
config_opts['plugin_conf']['tmpfs_enable'] = """ + str(tmpfs) + """
config_opts['plugin_conf']['tmpfs_opts'] = {}
config_opts['plugin_conf']['tmpfs_opts']['required_ram_mb'] = 1024
config_opts['plugin_conf']['tmpfs_opts']['max_fs_size'] = '768m'
config_opts['plugin_conf']['tmpfs_opts']['mode'] = '0755'
config_opts['plugin_conf']['tmpfs_opts']['keep_mounted'] = False
config_opts['plugin_conf']['sign_enable'] = """ + str(sign) + """
config_opts['plugin_conf']['sign_opts'] = {}
config_opts['plugin_conf']['sign_opts']['cmd'] = 'rpmsign'
config_opts['plugin_conf']['sign_opts']['opts'] = '--addsign %(rpms)s -D "%%_gpg_name """ + str(gpg_name) + """" -D "%%_gpg_path """ + str(gpg_path) + """'
"""
        return config

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

class MockChain(Mock):
    """
    Rebuild a bunch of source RPMs with mockchain.
    """

    name = "mockchain"

    description = ["mockchain"]
    descriptionDone = ["mockchain complete"]

    def __init__(self, localrepo=None, recursive=False, srpms=[], **kwargs):
        Mock.__init__(self, **kwargs)

        if not srpms or len(srpms) == 0:
            config.error("Please specify a list of srpms")

        self.command = ["/usr/bin/mockchain", "--root", self.root,
                        "-m", "--resultdir=" + self.resultdir,
                        "--tmp_prefix=buildbot"]
        if localrepo:
            self.command += ["-l", localrepo]
        if recursive:
            self.command.append("--recurse")
        self.command += srpms

    def start(self):
        # Observe mockchain logs
        if self.resultdir:
            for lname in self.mock_logfiles:
                self.logfiles[lname] = self.build.path_module.join(self.resultdir,
                                                                   lname)
        else:
            for lname in self.mock_logfiles:
                self.logfiles[lname] = lname
        self.addLogObserver("state.log", MockChainStateObserver())

        # Remove old logs
        cmd = remotecommand.RemoteCommand("rmdir", {"dir":
                                                    map(lambda l: self.build.path_module.join("build", self.logfiles[l]),
                                                        self.mock_logfiles)})
        d = self.runCommand(cmd)

        @d.addCallback
        def removeDone(cmd):
            ShellCommand.start(self)
        d.addErrback(self.failed)
