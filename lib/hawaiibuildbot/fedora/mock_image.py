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

from mock import Mock

class FlattenKickstart(ShellCommand):
    """
    Flatten a kickstart file.
    """

    name = "ksflatten"

    description = ["flattening kickstart"]
    descriptionDone = ["kickstart flattened"]

    filename = None

    def __init__(self, filename=None, **kwargs):
        ShellCommand.__init__(self, **kwargs)

        self.filename = filename
        if not self.filename:
            config.error("You must specify a kickstart file name")

        self.command = ["ksflatten", "-c", self.filename, "-o", "flattened.ks"]

class CopyKickstart(ShellCommand):
    """
    Copy kickstart file to mock chroot.
    """

    name = "copy-kickstart"

    description = ["copying kickstart"]
    descriptionDone = ["kickstart copied"]

    def __init__(self, **kwargs):
        ShellCommand.__init__(self, **kwargs)

    def start(self):
        mock_config = self.getProperty("mock_config")
        self.command = ["mock", "-r", mock_config, "--no-cleanup-after",
                        "--disable-plugin=tmpfs", "--copyin", "flattened.ks", "/tmp/flattened.ks"]
        ShellCommand.start(self)

class RemoveKickstart(ShellCommand):
    """
    Remove kickstart file from mock chroot.
    """

    name = "rm-kickstart"

    description = ["removing kickstart"]
    descriptionDone = ["kickstart removed"]

    def __init__(self, **kwargs):
        ShellCommand.__init__(self, **kwargs)

    def start(self):
        mock_config = self.getProperty("mock_config")
        self.command = ["mock", "-r", mock_config, "--disable-plugin=tmpfs",
                        "--chroot", "rm", "-f", "/tmp/flattened.ks"]
        ShellCommand.start(self)

class CreateMockConfig(ShellCommand):
    """
    Create a custom mock configuration for image creation
    in a chroot.
    """

    name = "mockconfig"

    description = ["creating mock config"]
    descriptionDone = ["mock config created"]

    arch = None
    distro = None

    dstconfig = None

    def __init__(self, arch=None, distro=None, **kwargs):
        ShellCommand.__init__(self, **kwargs)

        self.arch = arch
        if not self.arch:
            config.error("You must specify the architecture")
        self.distro = distro
        if not self.distro:
            config.error("You must specify the distribution")

        root = "fedora-{}-{}".format(self.distro, self.arch)
        srcconfig = "/etc/mock/{}.cfg".format(root)
        self.dstconfig = "image-{}-{}.cfg".format(self.distro, self.arch)

        self.command = ["../helpers/mock-config",
                        "--srcconfig=" + srcconfig,
                        "--dstconfig=" + self.dstconfig,
                        "--profile=image"]

    def start(self):
        self.setProperty("mock_config", self.dstconfig, "CreateMockConfig")
        ShellCommand.start(self)

class MockLiveCd(Mock):
    """
    Create a live CD from a mock chroot.
    """

    name = "livecd-creator"

    description = ["creating livecd"]
    descriptionDone = ["livecd created"]

    arch = None
    distro = None
    title = None
    product = None
    imgname = None
    version = None

    def __init__(self, arch=None, distro=None, title=None,
                 product=None, imgname=None, version=None, **kwargs):
        Mock.__init__(self, root="-", resultdir="build", **kwargs)

        self.arch = arch
        if not self.arch:
            config.error("You must specify the architecture")
        self.distro = distro
        if not self.distro:
            config.error("You must specify the distribution")
        self.title = title
        if not self.title:
            config.error("You must specify a title")
        self.product = product
        if not self.product:
            config.error("You must specify the product name")
        self.imgname = imgname
        if not self.imgname:
            config.error("You must specify the imgname")
        self.version = version
        if not self.version:
            config.error("You must specify the version")

    def start(self):
        # We set the fs label to the same as the isoname if it exists,
        # taking at most 32 characters
        filename = "{}-{}-{}".format(self.imgname, self.version, self.arch)[:32]

        # Mock configuration
        mock_config = self.getProperty("mock_config")

        # Command
        cmd = ["/usr/bin/livecd-creator", "--releasever=" + self.distro,
               "--title=" + self.title, "--product=" + self.product,
               "-c", "flattened.ks", "-f", filename, "-d", "-v",
               "--cache", "/tmp/buildbot-livecd"]
        self.command = ["mock", "-r", mock_config, "--resultdir=" + self.resultdir,
                        "--arch=" + self.arch, "--disable-plugin=tmpfs",
                        "--cwd", "/tmp", "--chroot", "--"] + cmd
        Mock.start(self)
