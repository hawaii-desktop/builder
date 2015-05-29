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

from buildbot.steps.package.rpm.mock import MockBuildSRPM, MockRebuild

class BuildSourcePackage(MockBuildSRPM):
    """
    Create a SRPM from sources using mock.
    """

    def __init__(self, arch, distro, pkgname, **kwargs):
        self.arch = arch
        self.distro = distro
        self.pkgname = pkgname
        root = "fedora-{}-{}".format(self.distro, self.arch)
        resultdir = "../results"
        spec = "{}.spec".format(self.pkgname)
        MockBuildSRPM.__init__(self, root=root, resultdir=resultdir, spec=spec, **kwargs)
        self.name = "srpm {} {}/{}".format(self.pkgname, self.distro, self.arch)

class BuildBinaryPackage(MockRebuild):
    """
    Build a SRPM using mock.
    """

    def __init__(self, arch, distro, pkgname, **kwargs):
        self.arch = arch
        self.distro = distro
        self.pkgname = pkgname
        root = "fedora-{}-{}".format(self.distro, self.arch)
        resultdir = "../results"
        pkg_info = self.getProperty("package_info")
        if len(pkg_info.get(self.pkgname, {}).keys()) == 0:
            config.error("Unable to retrieve package information")
        srpm = package_info[self.pkgname]["srpm"]
        MockRebuild.__init__(self, root=root, resultdir=resultdir, srpm=srpm, **kwargs)
        self.name = "rpm {} {}/{}".format(self.pkgname, self.distro, self.arch)
