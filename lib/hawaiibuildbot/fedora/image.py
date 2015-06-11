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
        cmd = ["livecd-creator", "--releasever=" + self.distro,
               "--title=" + self.title, "--product=" + self.product,
               "-c", "flattened.ks", "-f", filename, "-d", "-v",
               "--cache", "../cache"]
        self.command = ["sudo",] + cmd

class CreateAppliance(ShellCommand):
    """
    Create an appliance.
    """

    name = "appliance-creator"

    description = ["creating appliance"]
    descriptionDone = ["appliance created"]

    haltOnFailure = True
    flunkOnFailure = True

    renderables = ["resultdir",]

    logfilename = "appliance.log"

    resultdir = "../results"
    cachedir = "../cache"

    arch = None
    distro = None
    title = None
    version = None

    def __init__(self, arch=None, distro=None, title=None, version=None, **kwargs):
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
        self.version = version
        if not self.version:
            config.error("You must specify the version")

    def start(self):
        # Observe log
        self.logfiles[self.logfilename] = \
            self.build.path_module.join(self.resultdir, self.logfilename)

        # Remove old log
        cmd = remotecommand.RemoteCommand("rmdir", {"dir": self.logfiles[self.logfilename]})
        d = self.runCommand(cmd)

        # Command
        resultdir = self.build.path_module.abspath(self.resultdir)
        cachedir = self.build.path_module.abspath(self.cachedir)
        name = "{}-{}-{}".format(self.title, self.version, self.arch)
        cmd = ["appliance-creator", "--logfile", self.logfiles[self.logfile], "--cache", cachedir,
               "-d", "-v", "-o", resultdir, "--format=raw", "--checksum",
               "--name", name, "--version", self.distro, "--release", self.version,
               "-c", "flattened.ks"]
        self.command = ["sudo",] + cmd

        @d.addCallback
        def removeDone(cmd):
            ShellCommand.start(self)
        d.addErrback(self.failed)
