#!/bin/bash
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

# Base docker image
from fedora:22
maintainer Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>

# Update
run dnf update -y

# Install packages
run dnf install -y make gcc git sudo openssh-server
run dnf install -y python-devel python-pip
run dnf install -y supervisor
run dnf install -y rsync

# Install mock and rpmbuild
run dnf install -y mock rpm-build

# Install live CD and appliance tools
run dnf install -y livecd-tools appliance-tools fedora-kickstarts spin-kickstarts

# Install repository tools
run dnf install -y createrepo

# Create data area
run mkdir -p /data/buildbot

# Install Python modules
run pip install --upgrade pip
run pip install mock pyaml networkx twisted

# Install buildbot
run cd /data/buildbot; \
    git clone https://github.com/buildbot/buildbot src
run cd /data/buildbot/src/slave; python setup.py install

# Create buildbot configuration
run cd /data/buildbot; \
    buildslave create-slave slave @MASTER@ @SLAVE@ password

# Create buildbot service
run /bin/echo -e "\
[program:buildslave]\n\
command=twistd --nodaemon --no_save -y buildbot.tac\n\
directory=/data/buildbot/slave\n\
environment=LC_ALL=\"C\"\n" > \
    /etc/supervisord.d/buildbot.ini

# Run supervisord
cmd ["/usr/bin/supervisord", "-n"]
