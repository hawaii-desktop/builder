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

# Based on buildbot Dockerfile made by Daniel Mizyrycki <daniel@dotcloud.com>
#
# Copyright Buildbot Team Members

# Base docker image
from base/archlinux:latest
maintainer Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>

# Update
run pacman -Sy
run pacman -S --noconfirm archlinux-keyring
run pacman-key --populate archlinux
run pacman -Syu --noconfirm
run pacman-db-upgrade

# Install build tools and supervisor
run pacman -S --noconfirm base-devel git sudo openssh python2 python2-pip supervisor

# Set ssh superuser (username: admin   password: admin)
run mkdir /data
run useradd -m -d /data/buildbot -p sa1aY64JOY94w admin
run sed -Ei 's/adm:x:4:/admin:x:4:admin/' /etc/group
run sed -Ei 's/(\%admin ALL=\(ALL\) )ALL/\1 NOPASSWD:ALL/' /etc/sudoers

# Install Python modules
run pip2 install --upgrade pip
run pip2 install mock pyaml networkx twisted requests txrequests

# Install buildbot
run cd /data/buildbot; sudo -u admin sh -c \
    "git clone https://github.com/buildbot/buildbot src"
run cd /data/buildbot/src/slave; python2 setup.py install

# Create buildbot configuration
run cd /data/buildbot; sudo -u admin sh -c \
    "buildslave create-slave slave @MASTER@ @SLAVE@ password"

# Set buildbot and ssh services
run mkdir -p /etc/supervisor/conf.d
run /bin/echo -e "[program:sshd]\ncommand=/usr/sbin/sshd -D\n" > \
    /etc/supervisor/conf.d/sshd.conf
run /bin/echo -e "\
[program:buildslave]\n\
command=twistd --nodaemon --no_save -y buildbot.tac\n\
directory=/data/buildbot/slave\n\
user=admin\n" > \
    /etc/supervisor/conf.d/buildbot.conf

# Expose container port 22 to a random port in the host
expose 22

# Run supervisord
cmd ["/usr/bin/supervisord", "-n"]
