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

# Update
run pacman -Syu --noconfirm

# Install buildbot and its dependencies
run pacman -S --noconfirm base base-devel git sudo openssh python2-pip
run pip2 install buildbot buildbot_slave

# Set ssh superuser (username: admin   password: admin)
run mkdir /data
run useradd -m -d /data/buildbot -p sa1aY64JOY94w admin
run sed -Ei 's/adm:x:4:/admin:x:4:admin/' /etc/group
run sed -Ei 's/(\%admin ALL=\(ALL\) )ALL/\1 NOPASSWD:ALL/' /etc/sudoers

# Create buildbot configuration
run cd /data/buildbot; sudo -u admin sh -c \
    "buildslave create-slave slave localhost:9989 slave1 password"

# Enable ssh
run mkdir -p /etc/systemd/system/multi-user.target.wants
run ln -s /usr/lib/systemd/system/sshd.service /etc/systemd/system/multi-user.target.wants/sshd.service

# Set buildbot master service and enable it
run /bin/echo -e "\
[Unit]\n\
Description=Buildbot Master\n\
\n\
[Service]\n\
ExecStart=twistd --nodaemon --no_save -y /data/buildbot/slave/buildbot.tac\n\
User=admin\n" > \
    /etc/systemd/system/multi-user.target.wants/buildbot-slave.service

# Expose container port 22 to a random port in the host
expose 22

# Run supervisord
#cmd ["/usr/bin/supervisord", "-n"]
cmd ["/sbin/init"]
