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
from buildbot.process import logobserver
from buildbot.steps.shell import ShellCommand

class SRPMBuild(ShellCommand):
    """
    Build a SRPM.
    """

    name = "srpmbuilder"

    _srpm = None

    def __init__(self, specfile=None, topdir="`pwd`", builddir="`pwd`",
                 rpmdir="`pwd`", sourcedir="`pwd`", specdir="`pwd`",
                 srcrpmdir="`pwd`", vcsRevision=False, **kwargs):
        ShellCommand.__init__(self, **kwargs)

        self.specfile = specfile
        if not self.specfile:
            config.error("You must specify a specfile")

        self.vcsRevision = vcsRevision

        self.command = 'rpmbuild ' \
            '--define "_topdir %s" --define "_builddir %s" ' \
            '--define "_rpmdir %s" --define "_sourcedir %s" ' \
            '--define "_specdir %s" --define "_srcrpmdir %s"' % \
            (topdir, builddir, rpmdir, sourcedir, specdir, srcrpmdir)

        self.addLogObserver("stdio", logobserver.LineConsumerLogObserver(self.logConsumer))

    def start(self):
        if self.vcsRevision:
            date = self.getProperty("got_date")
            revision = self.getProperty("got_shortrev")
            checkout = "{}git{}".format(date, revision)
            self.command += ' --define "_checkout %s"' % checkout
        self.command += " -bs " + self.specfile
        ShellCommand.start(self)

    # Read-only properties
    srpm = property(lambda self: self._srpm)

    def logConsumer(self):
        self.loglines = []
        self.errors = []

        # Errors start with one of these prefixes
        errors = ["   ", "RPM build errors:", "error: "]

        import re
        r = re.compile(r"^Wrote: .*/([^/]*.src.rpm)")
        while True:
            stream, line = yield
            m = r.search(line)
            if m:
                self._srpm = m.group(1)
                self.setProperty("srpm", self._srpm, "SRPMBuild")
            else:
                for prefix in errors:
                    if line.startswith(prefix):
                        self.errors.append(line)
                        break

    def createSummary(self, log):
        self.addCompleteLog("SRPM Build Log", "".join(self.loglines))
        if len(self.errors) > 0:
            self.addCompleteLog("SRPM Errors", "".join(self.errors))
