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

from buildbot.steps.shell import ShellCommand

import shell
from buildsteps import *

class PrepareSources(ShellCommand):
    """
    Create a tarball of the upstream sources and create the spec file.
    """

    def __init__(self, pkgname, **kwargs):
        ShellCommand.__init__(self, **kwargs)
        self.command = ["../../helpers/mksrc", pkgname]
        self.name = "prepare {}".format(pkgname)

class SourcesScan(shell.ShellCommand):
    """
    Scans all the spec files to retrieve NVR, provides and requires.
    """

    name = "sourcesscan"

    pkgnames = None

    def __init__(self, pkgnames, arch, distro, **kwargs):
        shell.ShellCommand.__init__(self, **kwargs)
        self.pkgnames = pkgnames
        self.arch = arch
        self.distro = distro

    @defer.inlineCallbacks
    def run(self):
        log = yield self.addLog("logs")

        # Make a list with package information
        pkg_info = []
        for pkgname in self.pkgnames:
            path = "../{}/work/{}.spec".format(pkgname, pkgname)
            # Retrieve NVR
            cmd = yield self._makeCommand(["../../helpers/spec-nvr", path])
            yield self.runCommand(cmd)
            if cmd.didFail():
                yield log.addStderr(u"Unable to determine NVR from \"{}\"\n".format(path))
                defer.returnValue(FAILURE)
            n, e, v, r = cmd.stdout.strip().split(" ")
            # Retrieve provides
            cmd = yield self._makeCommand(["../../helpers/spec-provides", path])
            yield self.runCommand(cmd)
            if cmd.didFail():
                yield log.addStderr(u"Unable to determine provides from \"{}\"\n".format(path))
                defer.returnValue(FAILURE)
            provides = cmd.stdout.strip().split("\n")
            # Retrieve requires
            cmd = yield self._makeCommand(["../../helpers/spec-requires", path])
            yield self.runCommand(cmd)
            if cmd.didFail():
                yield log.addStderr(u"Unable to determine requires from \"{}\"\n".format(path))
                defer.returnValue(FAILURE)
            provides = cmd.stdout.strip().split("\n")
            # Populate list
            pkg_info.append({"name": n, "epoch": e, "version": v, "release": r,
                             "provides": provides, "requires": requires})
        self.setProperty("pkg_info", pkg_info, "SourcesScan")
        yield log.addStdout(u"Package information: {}\n".format(pkg_info))

        # Update the list of dependencies removing those provided by the system
        for pkg in pkg_info:
            for dep in pkg["requires"]:
                providers = [npkg for npkg in pkg_info if dep in npkg["provides"] or npkg["name"] == dep]
                if len(provides) > 0:
                    deps.append(provides[0]["name"])
            pkg["requires"] = deps

        # Sort packages based on their dependencies
        names = [pkg["name"] for pkg in pkg_info]
        graph = nx.DiGraph()
        for pkg in pkg_info:
            if len(pkg["requires"]) == 0:
                graph.add_node(pkg["name"])
            else:
                for dep in pkg["requires"]:
                    if dep != pkg["name"]:
                        graph.add_edge(dep, pkg["name"])
        sorted_names = nx.topological_sort(graph)
        yield log.addStdout(u"Sorted packages:\n\t{}\n".format("\n\t".join(sorted_names)))

        # Remove existing packages from the build list
        existing_packages = self.getProperty("existing_packages", [])
        for pkg in existing_packages:
            # Check whether the spec file has a higher version
            spec_info = pkg_info[names.index(pkg["name"])]
            evr1 = "{}:{}-{}".format(spec_info["epoch"], spec_info["version"], spec_info["release"])
            evr1 = "{}:{}-{}".format(pkg["epoch"], pkg["version"], pkg["release"])
            cmd = yield self._makeCommand(["../../helpers/needs-rebuild", evr1, evr2])
            yield self.runCommand(cmd)
            if cmd.didFail():
                defer.returnValue(FAILURE)
            if cmd.stdout.strip() == "no":
                yield log.addStdout(u"Package \"{}\" has already been built, skipping...\n".format(pkgname))
                sorted_names.remove(pkgname)

        # Add build steps for all the packages
        steps = []
        for name in sorted_names:
            info = pkg_info[names.index(name)]
            steps.append(BuildSourcePackage(arch=self.arch, distro=self.distro,
                                            pkgname=name, workdir="{}/work".format(pkgname)))
            steps.append(BuildBinaryPackage(arch=self.arch, distro=self.distro,
                                            pkgname=name, workdir="{}/work".format(pkgname)))
        self.build.addStepsAfterCurrentStep(steps)
