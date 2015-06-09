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
from buildbot.process.buildstep import ShellMixin
from buildbot.plugins import steps
from buildbot.status.results import SUCCESS, FAILURE

from twisted.internet import defer

from buildbot.steps.package.rpm.mock import Mock

class MockRebuild(Mock):
    """
    Custom Mock step that rebuild the SRPM.
    """

    def __init__(self, repodir=None, vcsRevision=False, **kwargs):
        Mock.__init__(self, **kwargs)
        self.repodir = repodir
        self.vcsRevision = vcsRevision

    def start(self):
        if self.repodir:
            self.command = ["mockchain", "--root", self.root]
            if self.resultdir:
                self.command += ["-m", "--resultdir=" + self.resultdir]

            builddir = self.getProperty("builddir")
            relpath = self.build.path_module.join(builddir, self.repodir)
            abspath = self.build.path_module.abspath(relpath)
            self.command += ["-a", "file://" + abspath]

        if self.vcsRevision:
            date = self.getProperty("got_date")
            revision = self.getProperty("got_shortrev")
            if self.repodir:
                self.command += ["-m", "--define=_checkout %sgit%s" % (date, revision)]
            else:
                self.command += ["--define", "_checkout %sgit%s" % (date, revision)]

        for k in ("vendor", "packager", "distribution"):
            if self.repodir:
                self.command += ["-m", "--define=%s Hawaii" % k]
            else:
                self.command += ["--define", "%s Hawaii" % k]

        srpm = self.getProperty("srpm")
        if self.repodir:
            self.command.append(srpm)
        else:
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

class BuildNeeded(ShellMixin, steps.BuildStep):
    """
    Determine whether we previously built the latest package version.
    """

    name = "build-needed"

    def __init__(self, repodir=None, **kwargs):
        kwargs = self.setupShellMixin(kwargs, prohibitArgs=["command"])
        steps.BuildStep.__init__(self, haltOnFailure=True, **kwargs)

        self.repodir = repodir
        if not self.repodir:
            config.error("You must specify a repository directory")

    @defer.inlineCallbacks
    def run(self):
        log = yield self.addLog("logs")

        # Determine NEVR from spec file
        srpm = self.getProperty("srpm")
        cmd = yield self._makeCommand(["../helpers/needs-rebuild", srpm, self.repodir])
        yield self.runCommand(cmd)
        if cmd.didFail():
            yield log.addStderr(u"Unable to determine whether {} will be built\n".format(srpm))
            defer.returnValue(FAILURE)
        result = cmd.stdout.strip()
        if result not in ("yes", "no"):
            yield log.addStderr(u"Unable to determine whether {} will be built\n".format(srpm))
            defer.returnValue(FAILURE)
        self.setProperty("build_needed", result == "yes", "BuildNeeded")
        defer.returnValue(SUCCESS)

    def _makeCommand(self, args, **kwargs):
        return self.makeRemoteShellCommand(collectStdout=True, collectStderr=True,
            command=args, **kwargs)

def isBuildNeeded(step):
    return step.build.getProperty("build_needed")
