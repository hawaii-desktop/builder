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
