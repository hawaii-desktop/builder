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

from buildbot.steps.shell import ShellCommand

import shell

class CreateRepo(ShellCommand):
    """
    Creates or updates the repository.
    """

    name = "createrepo"

    repodir = None

    def __init__(self, repodir, **kwargs):
        ShellCommand.__init__(self, **kwargs)
        self.repodir = repodir
        self.command = ["../../helpers/createrepo", self.repodir)

class RepositoryScan(shell.ShellCommand):
    """
    Scans a repository to find which packages have already been built.
    """

    name = "reposcan"

    repodir = None

    def __init__(self, repodir, **kwargs):
        shell.ShellCommand.__init__(self, **kwargs)
        self.repodir = repodir

    @defer.inlineCallbacks
    def run(self):
        log = yield self.addLog("logs")

        # Make a list of packages that have been built already
        cmd = yield self._makeCommand(["/usr/bin/find", self.repodir, "-type", "f", "-name", "*.src.rpm"])
        yield self.runCommand(cmd)
        if cmd.didFail():
            defer.returnValue(FAILURE)
        filenames = cmd.stdout.strip().split("\n")
        yield log.addStdout(u"Existing SRPMs: {}\n".format(filenames))

        # Turn the list of file names into a list of NVRs
        existing_packages = []
        for path in filenames:
            cmd = yield self._makeCommand(["../../helpers/srpm-nvr", path])
            yield self.runCommand(cmd)
            if cmd.didFail():
                yield log.addStderr(u"Unable to determine NVR for \"{}\"\n".format(path))
                defer.returnValue(FAILURE)
            n, e, v, r = cmd.stdout.strip().split(" ")
            existing_packages.append({"name": n, "epoch": e, "version": v, "release": r})
        self.setProperty("existing_packages", existing_packages, "RepositoryScan")
        yield log.addStdout(u"Existing packages: {}\n".format(existing_packages))

        defer.returnValue(SUCCESS)
