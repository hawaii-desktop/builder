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

from repo import *
from sources import *
from buildsteps import *

class CiFactory(BuildFactory):
    """
    Factory to build a repository of the Hawaii CI for a certain architecture.
    Strategoy:
        - Copy all the helpers from master to slave
        - Create or update local repository
        - Clone upstream and downstream sources and prepare the
          spec file and source tarball
        - Scan the repository to find packages that have already been built
        - Scan the sources to retrieve package information
        - Determine which packages need to be built and order the list
          based on dependencies
        - Build SRPMs
        - Build RPMs
    """

    def __init__(self, sources, arch):
        BuildFactory.__init__(self, [])

        # Copy helpers
        for helper in ("createrepo", "mksrc", "srpm-nvr", "spec-nvr",
                       "spec-provides", "spec-requires", "needs-rebuild"):
            self.addStep(steps.FileDownload(name="helper " + helper,
                                            mastersrc="helpers/fedora/" + helper,
                                            slavedest="../helpers/" + helper,
                                            mode=0755))

        # Create or update local repository
        self.addStep(CreateRepo(repodir="repository"))

        # Clone all sources and prepare the spec file and source tarball
        for pkgname in sources.keys():
            self.addStep(Git(name="{} upstream".format(pkgname), repourl=sources[pkgname]["upstreamsrc"],
                             mode="incremental", workdir="{}/upstream".format(pkgname)))
            self.addStep(Git(name="{} downstream".format(pkgname), repourl=sources[pkgname]["downstreamsrc"],
                             mode="incremental", workdir="{}/downstream".format(pkgname)))
            self.addStep(PrepareSources(pkgname, workdir=pkgname))

        # Scan the repository to find which packages have already been built
        self.addStep(RepositoryScan(repodir="../repository"))

        # Scan sources and build
        self.addStep(SourcesScan(pkgnames=sources.keys(), arch=arch, distro="22"))
