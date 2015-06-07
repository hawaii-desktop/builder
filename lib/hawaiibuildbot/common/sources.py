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

from buildbot.status.results import SUCCESS
from buildbot.steps.source.git import Git as BuildBotGit

from twisted.internet import defer
from twisted.python import log

class Git(BuildBotGit):
    """
    Custom git build step that saves the git short revision and date.
    The properties are:
      - got_date: Short date of the last commit
      - got_revision: Short revision
    """

    def __init__(self, **kwargs):
        BuildBotGit.__init__(self, **kwargs)

    def startVC(self, branch, revision, patch):
        d = BuildBotGit.startVC(self, branch, revision, patch)
        d.addCallback(self.parseGotDate)
        d.addCallback(self.parseGotShortRev)
        return

    @defer.inlineCallbacks
    def parseGotDate(self, _=None):
        # git log -1 --format="%cd" --date=short | tr -d '-'
        cmd = ["log", "-1", '--format="%cd"', "--date=short"]
        stdout = yield self._dovccmd(cmd, collectStdout=True)
        value = stdout.strip().replace('-', '').replace('"', '')
        log.msg(u"Got Git date: {}".format(value))
        self.updateSourceProperty("got_date", value)
        defer.returnValue(SUCCESS)

    @defer.inlineCallbacks
    def parseGotShortRev(self, _=None):
        # git log -1 --format="%h"
        cmd = ["log", "-1", '--format="%h"']
        stdout = yield self._dovccmd(cmd, collectStdout=True)
        value = stdout.strip().replace('-', '').replace('"', '')
        log.msg(u"Got Git short rev: {}".format(value))
        self.updateSourceProperty("got_shortrev", value)
        defer.returnValue(SUCCESS)
