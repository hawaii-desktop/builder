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

from buildbot import config
from buildbot.steps.shell import ShellCommand

from buildbot.steps.package.rpm.mock import Mock

class MockRebuild(Mock):
    """
    Custom Mock step that rebuild the SRPM.
    """

    def __init__(self, vcsRevision=False, **kwargs):
        Mock.__init__(self, **kwargs)
        self.vcsRevision = vcsRevision

    def start(self):
        if self.vcsRevision:
            date = self.getProperty("got_date")
            revision = self.getProperty("got_shortrev")
            self.command += ["--define", "_checkout %sgit%s" % (date, revision)]

        for k in ("vendor", "packager", "distribution"):
            self.command += ["--define", "{} Hawaii".format(k)]

        srpm = self.getProperty("srpm")
        self.command += ["--rebuild", srpm]

        Mock.start(self)

class TarXz(ShellCommand):
    """
    Create a tarball compressed with xz.
    """

    name = "tar"

    def __init__(self, filename=None, srcdir=None, **kwargs):
        ShellCommand.__init__(self, **kwargs)

        if not filename:
            config.error("You must specify a file name")
        if not srcdir:
            config.error("You must specify a source directory")

        self.command = ["tar", "-cJf", filename, srcdir]
