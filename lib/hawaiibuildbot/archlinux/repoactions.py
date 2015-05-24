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

import networkx as nx

from buildbot.process.buildstep import ShellMixin, BuildStep
from buildbot.status.results import *

from twisted.internet import defer

from pkgactions import BinaryPackageBuild

class RepositoryScan(ShellMixin, BuildStep):
    """
    Scans a repository to find packages and build them.
    """
    name = "scan-repository"
    description = "Scan a repository and build packages not yet built"
    packages = []

    def __init__(self, arch, channel, **kwargs):
        kwargs = self.setupShellMixin(kwargs, prohibitArgs=["command"])
        BuildStep.__init__(self, **kwargs)
        self.arch = arch
        self.channel = channel

    @defer.inlineCallbacks
    def run(self):
        log = yield self.addLog("logs")

        # Make a list of packages that have been built already
        cmd = yield self._makeRemoteCommand("ls built_packages")
        yield self.runCommand(cmd)
        if cmd.didFail():
            defer.returnValue(FAILURE)
        existing_packages = cmd.stdout.split()
        self.setProperty("existing_packages", existing_packages, "Repository Scan")

        # Find out which packages are meant for this channel
        data = self._loadYaml("buildinfo.yml")
        self.packages = data.get(self.channel, {}).get(self.arch, [])
        self.packages = ["hawaii-icon-themes-git",]
        if len(self.packages) == 0:
            yield log.addStdout("No packages to build found from the list")
            defer.returnValue(SKIPPED)

        # Get the dependencies and provides for the packages
        pkginfo = []
        for pkgname in self.packages:
            # Dependencies
            cmd = yield self._makeRemoteCommand("../helpers/pkgdepends {}/PKGBUILD".format(pkgname))
            yield self.runCommand(cmd)
            if cmd.didFail():
                defer.returnValue(FAILURE)
            depends = cmd.stdout.split()

            # Get the package names this package provides
            cmd = yield self._makeRemoteCommand("../helpers/pkgprovides {}/PKGBUILD".format(pkgname))
            yield self.runCommand(cmd)
            if cmd.didFail():
                defer.returnValue(FAILURE)
            provides = cmd.stdout.split()

            # Append package information
            pkg_info.append({
                "name": pkgname,
                "depends": depends,
                "provides": provides
            })

        # Update the list of dependencies removing dependencies provided by the system
        for pkg in pkg_info:
            deps = []
            for dep in pkg["depends"]:
                providers = [pkg for pkg in pkg_info if dep in pkg["provides"] or pkg["name"] == dep]
                if len(provides) > 0:
                    deps.append(providers[0]["name"])
            pkg["depends"] = deps

        # Sort packages based on their dependencies
        names = [pkg["name"] for pkg in pkg_info]
        graph = nx.DiGraph()
        for pkg in pkg_info:
            if len(pkg["depends"]) == 0:
                graph.add_node(pkg["name"])
            else:
                for dep in pkg["depends"]:
                    graph.add_edge(dep, pkg["name"])
        sorted_names = nx.topological_sort(graph)

        # Create build steps for the sorted packages list
        steps = []
        for name in sorted_names:
            info = pkg_info[names.index(name)]
            steps.append(BinaryPackageBuild(name=name, arch=self.arch,
                            depends=info["depends"], provides=info["provides"]))

        yield log.addStdout(u"Sorted packages: {}\n".format(sorted_names))

        self.build.addStepsAfterCurrentStep(steps)
        self.setProperty("packages", sorted_package_names, "Repository Scan")

        defer.returnValue(SUCCESS)

    def getCurrentSummary(self):
        return {"step": u"scanning repository"}

    def getResultSummary(self):
        return {"step": u"{} packages".format(len(self.packages))}

    def _makeRemoteCommand(self, cmd, **kwargs):
        args = cmd.split(" ")
        return self.makeRemoteShellCommand(collectStdout=True, collectStderr=True,
            command=args, **kwargs)

    def _loadYaml(self, fileName):
        from yaml import load
        try:
            from yaml import CLoader as Loader
        except ImportError:
            from yaml import Loader
        try:
            stream = open(fileName, "r")
        except:
            return {}
        return load(stream, Loader=Loader)
