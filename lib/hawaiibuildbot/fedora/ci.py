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

import os, re
import networkx as nx

from buildbot import config
from buildbot.steps.shell import ShellCommand
from buildbot.plugins import steps
from buildbot.process.buildstep import ShellMixin
from buildbot.status.results import *

from twisted.internet import defer

from mock import MockBuildSRPM, MockChain

class MakeSRPM(ShellCommand):
    """
    Clone upstream and downstream git repositories,
    create spec file and tarball, build the SRPM.
    """

    description = ["building srpm"]
    descriptionDone = ["srpm built"]

    def __init__(self, pkgname=None, repoinfo=None, **kwargs):
        ShellCommand.__init__(self, **kwargs)

        if not pkgname:
            config.error("You must specify a package name")
        if not repoinfo:
            config.error("You must specify repository information")

        self.name = "{} srpm".format(pkgname)

        repourl1 = repoinfo["upstream"]["repourl"]
        branch1 = repoinfo["upstream"].get("branch", "master")
        repourl2 = repoinfo["downstream"]["repourl"]
        branch2 = repoinfo["downstream"].get("branch", "master")

        self.command = ["../../helpers/make-srpm", pkgname,
                        repourl1, branch1, repourl2, branch2]

class BuildSourcePackages(ShellMixin, steps.BuildStep):
    """
    Collect SRPMs and start mockchain on them.
    """

    name = "buildsrpms"

    description = ["collecting srpms"]
    descriptionDone = ["srpms sent to mockchain"]

    # XXX: This won't totally work because some packages requires pkgconfig
    #      packages but we list actual package names only on provides.
    sort_by_deps = False

    def __init__(self, sources, arch, distro, **kwargs):
        kwargs = self.setupShellMixin(kwargs, prohibitArgs=["command"])
        steps.BuildStep.__init__(self, haltOnFailure=True, **kwargs)
        self.sources = sources
        self.arch = arch
        self.distro = distro

    @defer.inlineCallbacks
    def run(self):
        log = yield self.addLog("logs")

        if self.sort_by_deps:
            sources = self.sources
        else:
            from collections import OrderedDict
            sources = OrderedDict(sorted(self.sources.items(), key=lambda x: x[1]["ord"]))

        # Make a list with package information
        pkg_info = []
        for pkgname in sources.keys():
            # Spec file path
            path = "{}/work/{}.spec".format(pkgname, pkgname)
            # Retrieve NVR
            cmd = yield self._makeCommand(["../helpers/spec-nvr", path])
            yield self.runCommand(cmd)
            if cmd.didFail():
                yield log.addStderr(u"Unable to determine NVR from \"{}\"\n".format(path))
                defer.returnValue(FAILURE)
            n, e, v, r = cmd.stdout.strip().split(" ")
            # Retrieve provides
            cmd = yield self._makeCommand(["../helpers/spec-provides", path])
            yield self.runCommand(cmd)
            if cmd.didFail():
                yield log.addStderr(u"Unable to determine provides from \"{}\"\n".format(path))
                defer.returnValue(FAILURE)
            provides = []
            for i in cmd.stdout.split("\n"):
                provides.append(i.split(" = ")[0])
            # Retrieve requires
            cmd = yield self._makeCommand(["../helpers/spec-requires", path])
            yield self.runCommand(cmd)
            if cmd.didFail():
                yield log.addStderr(u"Unable to determine requires from \"{}\"\n".format(path))
                defer.returnValue(FAILURE)
            requires = []
            for i in cmd.stdout.split("\n"):
                requires.append(i.split(" ")[0])
            # Populate list
            pkg_info.append({"name": n, "epoch": e, "version": v, "release": r,
                             "provides": provides, "requires": requires,
                             "pkgname": pkgname, "pkgdir": pkgname + "/work"})

        # Update the list of dependencies removing those provided by the system
        for pkg in pkg_info:
            deps = []
            for dep in pkg["requires"]:
                providers = [npkg for npkg in pkg_info if dep in npkg["provides"] or npkg["name"] == dep]
                if len(providers) > 0:
                    deps.append(providers[0]["name"])
            pkg["requires"] = deps

        # Log package information
        buffer = ""
        for pkg in pkg_info:
            buffer += "\t- {} {}:{}-{}\n".format(pkg["name"], pkg["epoch"], pkg["version"], pkg["release"])
            if len(pkg["provides"]) > 0:
                buffer += "\t\t- Provides:\n"
                for provide in pkg["provides"]:
                    buffer += "\t\t\t- {}\n".format(provide)
            if len(pkg["requires"]) > 0:
                buffer += "\t\t- Requires:\n"
                for require in pkg["requires"]:
                    buffer += "\t\t\t- {}\n".format(require)
        self.setProperty("pkg_info", pkg_info, "MoveSourcePackages")
        yield log.addStdout(u"Package information:\n{}\n\n".format(buffer))

        # Find SRPMs
        srpms = []
        for pkg in pkg_info:
            # Find the artifacts
            cmd = yield self._makeCommand(["/usr/bin/find", pkg["pkgdir"], "-type", "f", "-name", "*.src.rpm"])
            yield self.runCommand(cmd)
            if cmd.didFail():
                defer.returnValue(FAILURE)

            # Add artifact to the list
            artifacts = cmd.stdout.strip().split(" ")
            buffer = ""
            for artifact in artifacts:
                buffer += "\t{}\n".format(artifact)
            yield log.addStdout(u"Artifacts for {}:\n{}\n".format(pkg["pkgname"], buffer))
            if pkg["epoch"] == "(none)":
                r = re.compile(r'.*\-{}\-{}\.src\.rpm'.format(pkg["version"], pkg["release"]))
            else:
                r = re.compile(r'.*\-{}:{}\-{}\.src\.rpm'.format(pkg["epoch"], pkg["version"], pkg["release"]))
            matching_artifacts = filter(r.match, artifacts)
            if len(matching_artifacts) == 0:
                yield log.addStderr(u"No matching artifacts found for {}\n".format(pkg["pkgname"]))
                defer.returnValue(FAILURE)
            buffer = ""
            for artifact in matching_artifacts:
                buffer += "\t{}\n".format(artifact)
            yield log.addStdout(u"Matching artifacts for {}:\n{}\n".format(pkg["pkgname"], buffer))

            # Add matching artifacts to the list
            pkg["srpms"] = matching_artifacts
            srpms += matching_artifacts

        # Sort package names by dependencies
        names = [pkg["name"] for pkg in pkg_info]
        if self.sort_by_deps:
            graph = nx.DiGraph()
            for pkg in pkg_info:
                if len(pkg["requires"]) == 0:
                    graph.add_node(pkg["name"])
                else:
                    for dep in pkg["requires"]:
                        if dep != pkg["name"]:
                            graph.add_edge(dep, pkg["name"])
            sorted_names = nx.topological_sort(graph)
        else:
            sorted_names = names
        yield log.addStdout(u"Sorted packages:\n\t- {}\n".format("\n\t- ".join(sorted_names)))

        # Sort SRPMs by dependencies
        sorted_srpms = []
        for name in sorted_names:
            for pkg in pkg_info:
                if pkg["name"] == name:
                    sorted_srpms += pkg["srpms"]
                    break

        # Log the list of sorted SRPMs
        yield log.addStdout(u"\nSRPMs:\n\t- {}\n".format("\n\t- ".join(sorted_srpms)))

        # Chain build
        root = "fedora-{}-{}".format(self.distro, self.arch)
        step = MockChain(root=root, recursive=True, srpms=sorted_srpms,
                         localrepo="../repository", resultdir="../results")
        self.build.addStepsAfterCurrentStep([step])

        defer.returnValue(SUCCESS)

    def _makeCommand(self, args, **kwargs):
        return self.makeRemoteShellCommand(collectStdout=True, collectStderr=True,
            command=args, **kwargs)

    @defer.inlineCallbacks
    def _runCommand(self, command, **kwargs):
        cmd = yield self._makeCommand(command, **kwargs)
        yield self.runCommand(cmd)
        defer.returnValue(not cmd.didFail())
