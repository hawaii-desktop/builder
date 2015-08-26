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
from buildbot.steps.source.git import Git
from buildbot.plugins import steps

from chrootactions import *
from repoactions import *
from sourceactions import *

class RepositoryFactory(BuildFactory):
    """
    Factory to build a repository of packages for a certain architecture.
    """

    def __init__(self, sources, arch):
        BuildFactory.__init__(self, sources)

        # Download the helpers
        for helper in ("pkgdepends", "pkgprovides", "pkgversion", "ccm-setup"):
            self.addStep(steps.FileDownload(name="helper " + helper,
                                            mastersrc="helpers/archlinux/" + helper,
                                            slavedest="../helpers/" + helper,
                                            mode=0755))
        # Create a directory to hold the packages that have been built
        self.addStep(steps.MakeDirectory(name="mkdir-repository", dir="repository"))
        # Create or update the chroot
        self.addStep(PrepareCcm(arch=arch))
        # Copy the list of packages to build from slave to master
        self.addStep(steps.FileUpload("buildinfo.yml", "tmp/buildinfo.yml", name="buildinfo-upload"))
        # Scan repository and find packages to build
        self.addStep(RepositoryScan(channel="ci", arch=arch))
        # Push back changes to version control
        if arch == "x86_64":
            self.addStep(PushSourceChanges())
