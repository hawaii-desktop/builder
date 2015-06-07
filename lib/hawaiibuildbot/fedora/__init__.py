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

class BasePackageFactory(BuildFactory):
    """
    Abstract class for all package factories.
    Provides basic configuration and steps for a local repository
    and upload to master.
    """

    def __init__(self, pkg, arch, distro, channel):
        BuildFactory.__init__(self, [])

        # Mock properties
        self.root = "fedora-{}-{}".format(distro, arch)
        self.resultdir = "../results"

        # Other properties
        self.repodir = "repository/{}/{}".format(channel, arch)
        self.arch = arch
        self.distro = distro
        self.channel = channel

        # Make sure the local repository is available
        self.addStep(ShellCommand(name="local repo",
                                  command="mkdir -p ../../%s/{noarch,source,%s}" % (self.repodir, arch)))

    def updateLocalRepository(self):
        # Update local repository
        for rpmset in (("src.rpm", "source"), ("noarch.rpm", "noarch"), (self.arch + ".rpm", self.arch)):
            dst = "../../{}/{}".format(self.repodir, rpmset[1])
            self.addStep(ShellCommand(name="mv " + rpmset[0],
                                      command="find %s -type f -name *.%s -exec mv -f {} %s \\;" % (self.resultdir, rpmset[0], dst)))
        self.addStep(ShellCommand(name="createrepo",
                                  command="createrepo -v --deltas --num-deltas 5 --compress-type xz ../../{}".format(self.repodir)))

    def uploadToMaster(self):
        # Update repository on master
        import datetime
        today = datetime.datetime.now().strftime("%Y%m%d")
        src = "../../{}".format(self.repodir)
        dst = "public_html/{}/{}/{}".format(self.channel, self.arch, today)
        self.addStep(steps.MasterShellCommand(name="clean repo", command="rm -rf " + dst))
        self.addStep(steps.DirectoryUpload(name="upload repo", compress="gz",
                                           slavesrc=src, masterdest=dst))
        self.addStep(steps.MasterShellCommand(name="repo permission",
                                              command="chmod a+rX -R " + dst))

class PackageFactory(BasePackageFactory):
    """
    Factory to build a single package.
    Logic:
      - Update sources
      - Build SRPM
      - Rebuild SRPM with mock
      - Update local repository
      - Upload repository to master
    """

    def __init__(self, pkg, arch, distro, channel):
        BasePackageFactory.__init__(self, pkg, arch, distro, channel)

        # Fetch sources
        self.addStep(Git(repourl=pkg["repourl"], branch=pkg.get("branch", "master"),
                         method="fresh", mode="full"))
        # Build SRPM
        self.addStep(rpmbuild.SRPMBuild(specfile=pkg["name"] + ".spec"))
        # Rebuild SRPM
        self.addStep(ci.MockRebuild(root=self.root, resultdir=self.resultdir))
        # Update local repository
        self.updateLocalRepository()
        # Update repository on master
        self.uploadToMaster()

class CiPackageFactory(BasePackageFactory):
    """
    Factory to build a single package from CI.
    Logic:
      - Update upstream sources
      - Update packaging sources
      - Create upstream tarball on the packaging directory
      - Build SRPM
      - Rebuild SRPM with mock
      - Update local repository
      - Upload repository to master
    """

    def __init__(self, pkg, arch, distro, channel):
        BasePackageFactory.__init__(self, pkg, arch, distro, channel)

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
        self.addStep(ci.MockRebuild(root=self.root, resultdir=self.resultdir,
                                    repodir="../" + self.repodir, vcsRevision=True))
        # Update local repository
        self.updateLocalRepository()
        # Update repository on master
        self.uploadToMaster()

class ImageFactory(BuildFactory):
    """
    Factory to spin images.
    Logic:
      - Clone repository
      - Flatten kickstart file
      - Run the creator over the flattened kickstart file
    """

    def __init__(self, repourl, arch, distro):
        BuildFactory.__init__(self, [])

        import datetime
        today = datetime.datetime.now().strftime("%Y%m%d")

        # Kickstart sources
        self.addStep(Git(repourl=repourl, method="fresh", mode="full"))
        # Flatten kickstart and create image
        if arch in ("i386", "x86_64"):
            self.addStep(image.FlattenKickstart(filename="hawaii-livecd.ks"))
            self.addStep(image.CreateLiveCd(arch=arch, distro=distro,
                                            title="Hawaii", product="Hawaii",
                                            imgname="hawaii", version=today))
        elif arch == "armhfp":
            self.addStep(image.FlattenKickstart(filename="hawaii-arm.ks"))
            self.addStep(image.CreateAppliance(arch=arch, distro=distro,
                                               title="Hawaii", version=today))
