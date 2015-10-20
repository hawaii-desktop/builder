#
# This file is part of Builder.
#
# Copyright (C) 2015 Pier Luigi Fiorini
#
# Author(s):
#    Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
#
# $BEGIN_LICENSE:AGPL3+$
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.
#
# $END_LICENSE$
#

# Base docker image
from fedora:22
maintainer Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>

# Update
run dnf update -y

# Install packages
run dnf install -y golang git make
run dnf install -y repoview createrepo

# Build binaries
env BUILDER_DIR /go/src/github.com/hawaii-desktop/builder
env GOPATH $BUILDER_DIR/Godeps/_workspace:/go
env PATH $BUILDER_DIR/bin:$PATH
workdir $BUILDER_DIR
copy . $BUILDER_DIR
copy deploy/docker/master/builder-master.ini /etc/builder/builder-master.ini
run make PREFIX=/go clean binaries

# Run
volume ["/var/log", "/var/cache/builder/master", "/srv/builder/repo"]
cmd ["builder-master"]
