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

from buildsteps import *

class CiFactory(BuildFactory):
    """
    Factory to build a repository of the Hawaii CI for a certain architecture.
    """

    def __init__(self, sources, arch):
        BuildFactory.__init__(self, [])

        # Copy helpers
        for helper in ("mksrc",):
            self.addStep(steps.FileDownload(name="helper " + helper,
                                            mastersrc="helpers/fedora/" + helper,
                                            slavedest="../helpers/" + helper,
                                            mode=0755))

        # Build the SRPM from git
        for pkgname in sources.keys():
            self.addStep(Git(name="{} upstream".format(pkgname), repourl=sources[pkgname]["upstreamsrc"], mode="incremental", workdir="sources/{}/{}".format(pkgname, pkgname)))
            self.addStep(Git(name="{} downstream".format(pkgname), repourl=sources[pkgname]["downstreamsrc"], mode="incremental", workdir="sources/{}/downstream".format(pkgname)))
            self.addStep(PrepareSources(pkgname, workdir="sources/{}".format(pkgname)))
            self.addStep(BuildSourcePackage(arch=arch, distro="22", pkgname=pkgname, workdir="sources/{}/work".format(pkgname)))
            self.addStep(AcquirePackageInfo(pkgname=pkgname, workdir="sources/{}/work".format(pkgname)))
            #self.addStep(BuildBinaryPackage(arch=arch, distro="22", pkgname=pkgname, workdir="sources/{}/work".format(pkgname)))
