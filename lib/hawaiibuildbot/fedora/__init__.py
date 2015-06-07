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

from buildbot.process.factory import BuildFactory
from buildbot.steps.source.git import Git
from buildbot.steps.shell import ShellCommand
from buildbot.plugins import steps

import ci
import image

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
        for helper in ("make-srpm", "spec-nvr", "spec-provides", "spec-requires"):
            self.addStep(steps.FileDownload(name="helper " + helper,
                                            mastersrc="helpers/fedora/" + helper,
                                            slavedest="../helpers/" + helper,
                                            mode=0755))

        # Build SRPMs
        for pkgname in sources.keys():
            self.addStep(ci.MakeSRPM(pkgname=pkgname, repoinfo=sources[pkgname],
                                     workdir="build/{}".format(pkgname)))

        # Chain build packages
        self.addStep(ci.BuildSourcePackages(sources=sources, arch=arch, distro=distro))

class ImageFactory(BuildFactory):
    """
    Factory to spin images.
    Logic:
      - Flatten kickstart file
      - Run the creator over the flattened kickstart file
    """

    def __init__(self, repourl, arch, distro):
        git = Git(repourl=repourl, mode="incremental", workdir="kickstarts")
        BuildFactory.__init__(self, [git])

        import datetime
        today = datetime.datetime.now().strftime("%Y%m%d")

        # Flatten kickstart and create image
        if arch in ("i386", "x86_64"):
            self.addStep(image.FlattenKickstart(filename="../kickstarts/hawaii-livecd.ks"))
            self.addStep(image.CreateLiveCd(arch=arch, distro=distro,
                                            title="Hawaii", product="Hawaii",
                                            imgname="hawaii", version=today))
        elif arch in ("armhfp",):
            self.addStep(image.FlattenKickstart(filename="../kickstarts/hawaii-arm.ks"))
            self.addStep(image.CreateAppliance(arch=arch, distro=distro,
                                               title="Hawaii", version=today))
