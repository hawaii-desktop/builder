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
from buildbot.steps.shell import ShellCommand
from buildbot.plugins import steps

import ci
import image
import rpmbuild

from hawaiibuildbot.common.sources import Git

class PackageFactory(BuildFactory):
    """
    Factory to build a single package.
    Logic:
      - Update sources
      - Build SRPM
      - Rebuild SRPM with mock
    """

    def __init__(self, pkg, arch, distro):
        BuildFactory.__init__(self, [])

        # Mock properties
        root = "fedora-{}-{}".format(distro, arch)
        resultdir = "../results"

        # Fetch sources
        self.addStep(Git(repourl=pkg["repourl"], branch=pkg.get("branch", "master"),
                         method="fresh", mode="full"))
        # Build SRPM
        self.addStep(rpmbuild.SRPMBuild(specfile=pkg["name"] + ".spec"))
        # Rebuild SRPM
        self.addStep(ci.MockRebuild(root=root, resultdir=resultdir))

class CiPackageFactory(BuildFactory):
    """
    Factory to build a single package from CI.
    Logic:
      - Update upstream sources
      - Update packaging sources
      - Create upstream tarball on the packaging directory
      - Build SRPM
      - Rebuild SRPM with mock
      - Update local repository
      - TODO: Update repository on master
    """

    def __init__(self, pkg, arch, distro, channel):
        BuildFactory.__init__(self, [])

        from buildbot.steps.package.rpm.mock import Mock

        # Mock properties
        root = "fedora-{}-{}".format(distro, arch)
        resultdir = "../results"

        # Other properties
        repodir = "repository/{}/{}".format(channel, arch)

        # Make sure the local repository is available
        self.addStep(ShellCommand(name="local repo",
                                  command="mkdir -p ../../%s/{noarch,source,%s}" % (repodir, arch)))

        # Fetch upstream sources
        self.addStep(Git(name="git upstream",
                         repourl=pkg["upstream"]["repourl"],
                         branch=pkg["upstream"].get("branch", "master"),
                         method="fresh", mode="full", workdir=pkg["name"]))
        # Fetch packaging sources
        self.addStep(Git(name="git packaging",
                         repourl=pkg["downstream"]["repourl"],
                         branch=pkg["downstream"].get("branch", "master"),
                         method="fresh", mode="full"))
        # Create sources tarball
        self.addStep(ci.TarXz(filename="{}.tar.xz".format(pkg["name"]),
                              srcdir="../" + pkg["name"]))
        # Build SRPM
        self.addStep(rpmbuild.SRPMBuild(specfile=pkg["name"] + ".spec",
                                        vcsRevision=True))
        # Rebuild SRPM
        self.addStep(ci.MockRebuild(root=root, resultdir=resultdir,
                                    repodir="../" + repodir, vcsRevision=True))
        # Update local repository
        for rpmset in (("src.rpm", "source"), ("noarch.rpm", "noarch"), ("%s.rpm" % arch, arch)):
            src = "{}/*.{}".format(resultdir, rpmset[0])
            dst = "../../{}/{}".format(repodir, rpmset[1])
            self.addStep(ShellCommand(name="mv " + rpmset[0],
                                      command=["bash", "-c", "[ -f {} ] && mv {} {} || exit 0".format(src, src, dst)]))
        self.addStep(ShellCommand(name="createrepo",
                                  command="createrepo -v --deltas --num-deltas 5 --compress-type xz ../../{}".format(repodir)))
        # TODO: Update repository on master

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
