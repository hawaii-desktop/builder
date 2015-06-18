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

    Steps performed by constructor:
      - Create directory for local repository
      - Copy helpers
    """

    def __init__(self, pkg, arch, distro, channel):
        BuildFactory.__init__(self, [])

        # Mock properties
        self.root = "fedora-{}-{}".format(distro, arch)
        self.resultdir = "../results"

        # Other properties
        self.repourl = "http://localhost:9999/{}/{}".format(channel, arch)
        self.reporootdir = "repository/fedora"
        self.repodir = "{}/{}".format(self.reporootdir, arch)
        self.arch = arch
        self.distro = distro
        self.channel = channel

        # Make sure the local repository is available
        self.addStep(ShellCommand(name="local repo",
                                  command="mkdir -p ../../%s/packages" % self.repodir))
        # Copy helpers
        for helper in ("needs-rebuild", "update-repo"):
            self.addStep(steps.FileDownload(name="helper " + helper,
                                            mastersrc="helpers/fedora/" + helper,
                                            slavedest="../helpers/" + helper,
                                            mode=0755))

    def updateLocalRepository(self):
        # Update local repository
        self.addStep(ShellCommand(name="mv src.rpm",
                                  command="find %s -type f -name *.src.rpm -exec mv -f {} ../../%s/source \\;" % (self.resultdir, self.reporootdir),
                                  doStepIf=ci.isBuildNeeded))
        self.addStep(ShellCommand(name="mv *.rpm",
                                  command="find %s -type f -name *.rpm -exec mv -f {} ../../%s/packages \\;" % (self.resultdir, self.repodir),
                                  doStepIf=ci.isBuildNeeded))
        self.addStep(ShellCommand(name="update-repo",
                                  command="../helpers/update-repo ../../{}".format(self.repodir)))

    def uploadSourcesToMaster(self):
        # Update repository on master
        import datetime
        today = datetime.datetime.now().strftime("%Y%m%d")
        src = "../../{}/source".format(self.reporootdir)
        dst = "public_html/fedora/{}/source".format(today)
        self.addStep(steps.MasterShellCommand(name="sources clear", command="rm -rf " + dst))
        self.addStep(steps.DirectoryUpload(name="sources upload", compress="gz",
                                           slavesrc=src, masterdest=dst))
        self.addStep(steps.MasterShellCommand(name="sources permission",
                                              command="find %s -type d -exec chmod -R u=rwx,g=rwx,o=rx {} \\;" % dst))

    def uploadBinariesToMaster(self):
        # Update repository on master
        import datetime
        today = datetime.datetime.now().strftime("%Y%m%d")
        src = "../../{}".format(self.repodir)
        dst = "public_html/fedora/{}/{}".format(today, self.arch)
        self.addStep(steps.MasterShellCommand(name="binaries clear", command="rm -rf " + dst))
        self.addStep(steps.DirectoryUpload(name="binaries upload", compress="gz",
                                           slavesrc=src, masterdest=dst))
        self.addStep(steps.MasterShellCommand(name="binaries permission",
                                              command="find %s -type d -exec chmod -R u=rwx,g=rwx,o=rx {} \\;" % dst))

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
        # Determine whether we need to build this package
        self.addStep(ci.BuildNeeded(specfile=pkg["name"] + ".spec",
                                    repodir="../../{}".format(self.reporootdir)))
        # Build SRPM
        self.addStep(ShellCommand(name="spectool",
                                  command="spectool -g -A {}.spec".format(pkg["name"])))
        self.addStep(rpmbuild.SRPMBuild(specfile=pkg["name"] + ".spec",
                                        doStepIf=ci.isBuildNeeded))
        # Rebuild SRPM
        self.addStep(ci.MockRebuild(root=self.root, resultdir=self.resultdir,
                                    repourl=self.repourl,
                                    doStepIf=ci.isBuildNeeded))
        # Update local repository
        self.updateLocalRepository()
        # Update repository on master
        self.uploadSourcesToMaster()
        self.uploadBinariesToMaster()

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

        # Fetch packaging sources
        self.addStep(Git(name="git packaging",
                         repourl=pkg["downstream"]["repourl"],
                         branch=pkg["downstream"].get("branch", "master"),
                         method="fresh", mode="full"))
        # Fetch upstream sources (must be after fetching the packaging so
        # that the last properties set by Git refers to upstream and can
        # be used later)
        self.addStep(Git(name="git upstream",
                         repourl=pkg["upstream"]["repourl"],
                         branch=pkg["upstream"].get("branch", "master"),
                         method="fresh", mode="full", workdir=pkg["name"]))
        # Determine whether we need to build this package
        self.addStep(ci.BuildNeeded(specfile=pkg["name"] + ".spec",
                                    repodir="../../{}".format(self.reporootdir)))
        # Create sources tarball
        self.addStep(ci.TarXz(filename="{}.tar.xz".format(pkg["name"]),
                              srcdir="../" + pkg["name"],
                              doStepIf=ci.isBuildNeeded))
        # Build SRPM
        self.addStep(rpmbuild.SRPMBuild(specfile=pkg["name"] + ".spec",
                                        vcsRevision=True,
                                        doStepIf=ci.isBuildNeeded))
        # Rebuild SRPM
        self.addStep(ci.MockRebuild(root=self.root, resultdir=self.resultdir,
                                    repourl=self.repourl, vcsRevision=True,
                                    doStepIf=ci.isBuildNeeded))
        # Update local repository
        self.updateLocalRepository()
        # Update repository on master
        self.uploadSourcesToMaster()
        self.uploadBinariesToMaster()

class RepositoryFactory(BuildFactory):
    """
    Factory to build a complete repository for a
    specific architecture.
    """

    def __init__(self, pkgs, arch):
        BuildFactory.__init__(self, [])

        for pkg in pkgs:
            trigger = "trigger-fedora-{}-{}".format(arch, pkg["name"])
            self.addStep(steps.Trigger(schedulerNames=[trigger],
                                       waitForFinish=True,
                                       updateSourceStamp=True))

class SyncFactory(BasePackageFactory):
    """
    Factory to copy packages to master.
    """

    def __init__(self, arch):
        BasePackageFactory.__init__(self, None, arch, None, None)

        # Update local repository
        self.updateLocalRepository()
        # Update repository on master
        self.uploadSourcesToMaster()
        self.uploadBinariesToMaster()

class ImageFactory(BuildFactory):
    """
    Factory to spin images.
    Logic:
      - Clone repository
      - Flatten kickstart file
      - Run the creator over the flattened kickstart file
    """

    def __init__(self, repourl=None, distro=None, profile=None, arch=None):
        BuildFactory.__init__(self, [])

        if not repourl:
            config.error("You must specify the repository URL")
        if not distro:
            config.error("You must specify the distro")
        if not profile:
            config.error("You must specify the profile")
        if not arch:
            config.error("You must specify the architecture")

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
