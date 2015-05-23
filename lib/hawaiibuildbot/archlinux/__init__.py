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
from buildbot.steps.shell import ShellCommand
from buildbot.steps.source.git import Git

from chrootactions import *
from repoactions import *

class RepositoryFactory(BuildFactory):
    """
    Factory to build a repository of packages for a certain architecture.
    """

    def __init__(self, source, arch):
        BuildFactory.__init__(self, [source])

        # Create a directory to hold the packages that have been built
        self.addStep(MkDirCommand("built_packages"))
        # Copy the helpers
        self.addStep(Git(repourl="git://github.com/hawaii-desktop/builder", mode="full", method="fresh", shallow=True))
        # Create or update the chroot
        self.addStep(MkDirCommand("chroot"))
        self.addStep(CreateOrUpdateChrootAction(arch=arch))
        # Scan repository and find packages to build
        self.addStep(RepositoryScan(channel="ci", arch=arch))

class MkDirCommand(ShellCommand):
    """
    Creates a directory.
    Nothing fancy but it shows the step name.
    """

    def __init__(self, dirname):
        ShellCommand.__init__(self, command="mkdir -p %s" % dirname)
        self.name = "make-directory {}".format(dirname)
