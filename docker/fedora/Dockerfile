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
run dnf update -y

# Install packages
run dnf install -y make gcc git sudo openssh-server
run dnf install -y python-devel python-pip
run dnf install -y supervisor

# Install mock and rpmbuild
run dnf install -y mock rpm-build

# Install live CD and appliance tools
run dnf install -y livecd-tools appliance-tools fedora-kickstarts spin-kickstarts

# Install repository tools
run dnf install -y createrepo

# User (username: admin   password: admin)
run mkdir /data
run useradd -m -d /data/buildbot -p sa1aY64JOY94w admin
run /bin/echo "%admin ALL=(ALL) ALL NOPASSWD: ALL" >>/etc/sudoers
run usermod -G mock -a admin

# Enable root and ssh access
run mkdir -p /run/sshd
run /bin/echo "root:password" | /usr/sbin/chpasswd
run /usr/bin/sed 's@session\s*required\s*pam_loginuid.so@session optional pam_loginuid.so@g' -i /etc/pam.d/sshd

# Install Python modules
run pip install --upgrade pip
run pip install mock pyaml networkx twisted

# Install buildbot
run cd /data/buildbot; sudo -u admin sh -c \
    "git clone https://github.com/buildbot/buildbot src"
run cd /data/buildbot/src/slave; python setup.py install

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
