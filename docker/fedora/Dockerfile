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
from fedora:latest
maintainer Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>

# Update
run yum update -y

# Install packages
run yum install -y make gcc git sudo openssh-server
run yum install -y python-devel python-pip

# Install mock
run yum install -y mock

# Set ssh superuser (username: admin   password: admin)
run mkdir /data
run useradd -m -d /data/buildbot -p sa1aY64JOY94w admin
run sed -Ei 's/adm:x:4:/admin:x:4:admin/' /etc/group
run sed -Ei 's/(\%admin ALL=\(ALL\) )ALL/\1 NOPASSWD:ALL/' /etc/sudoers

# Setup virtualenv
run pip install --upgrade pip
run pip install mock pyaml networkx twisted

# Install buildbot
run cd /data/buildbot; sudo -u admin sh -c \
    "git clone https://github.com/buildbot/buildbot src"
run cd /data/buildbot/src/slave; python setup.py install

# Create buildbot configuration
run cd /data/buildbot; sudo -u admin sh -c \
    "buildslave create-slave slave localhost:9989 @SLAVE@ password"

# Set buildbot master service and enable it
run /bin/echo -e "\
[Unit]\n\
Description=Buildbot Slave\n\
\n\
[Service]\n\
ExecStart=twistd --nodaemon --no_save -y /data/buildbot/slave/buildbot.tac\n\
User=admin\n" > \
    /etc/systemd/system/multi-user.target.wants/buildbot-slave.service

# Expose container port 22 to a random port in the host
expose 22

# Run slave
cmd ["sudo", "-u", "admin", "sh", "-c", "twistd --nodaemon --no_save -y /data/buildbot/slave/buildbot.tac"]
