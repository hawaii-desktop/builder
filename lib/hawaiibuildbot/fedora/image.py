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

from buildbot import config
from buildbot.steps.shell import ShellCommand

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

        self.command = ["ksflatten", "-c", self.filename, "-o", "/tmp/flattened.ks"]

class CreateLiveCd(ShellCommand):
    """
    Create a live CD.
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
        ShellCommand.__init__(self, **kwargs)

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
            config.error("You must specify the image name")
        self.version = version
        if not self.version:
            config.error("You must specify the version")

        # We set the fs label to the same as the isoname if it exists,
        # taking at most 32 characters
        filename = "{}-{}-{}".format(self.imgname, self.version, self.arch)[:32]

        # Command
        cmd = ["/usr/bin/livecd-creator", "--releasever=" + self.distro,
               "--title=" + self.title, "--product=" + self.product,
               "-c", "/tmp/flattened.ks", "-f", filename, "-d", "-v",
               "--cache", "/tmp/buildbot-livecd"]
        self.command = ["pkexec",] + cmd
