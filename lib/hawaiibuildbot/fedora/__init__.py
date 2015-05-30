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

from buildbot.process.factory import BuildFactory
from buildbot.steps.source.git import Git
from buildbot.steps.shell import ShellCommand
from buildbot.plugins import steps

import ci

class CiFactory(BuildFactory):
    """
    Factory to build a CI packages.
    Logic:
      - Copy all needed helpers from master to slave
      - For each package:
        - Clone sources
        - Prepare sources (spec file and tarball)
        - Build SRPM
      - Chain build all SRPMs
    """

    def __init__(self, sources, arch, distro):
        BuildFactory.__init__(self, [])

        # Copy helpers
        for helper in ("make-srpm", "spec-nvr"):
            self.addStep(steps.FileDownload(name="helper " + helper,
                                            mastersrc="helpers/fedora/" + helper,
                                            slavedest="../helpers/" + helper,
                                            mode=0755))

        # Build SRPMs
        for pkgname in sources.keys():
            self.addStep(ci.MakeSRPM(pkgname=pkgname, repoinfo=sources[pkgname],
                                     workdir="build/{}".format(pkgname)))

        # Chain build packages
        self.addStep(ci.BuildSourcePackages(pkgnames=sources.keys(), arch=arch,
                                            distro=distro))
