#!/usr/bin/env python2
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

from setuptools import setup, find_packages

setup(
    name="hawaiibuildbot",
    version="0.9.0",
    description="Hawaii Builder",
    long_description="Buildbot utilities for Hawaii.",
    url="http://hawaiios.org/",
    author="Pier Luigi Fiorini",
    author_email="pierluigi.fiorini@gmail.com",
    license="GPL3",
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "Topic :: Software Development :: Build Tools",
        "License :: OSI Approved :: GNU General Public License v3 (GPLv3)",
        "Programming Language :: Python :: 2",
        "Programming Language :: Python :: 2.6",
        "Programming Language :: Python :: 2.7",
    ],
    keywords="buildbot development",
    packages=["hawaiibuildbot", "hawaiibuildbot.archlinux",
              "hawaiibuildbot.common", "hawaiibuildbot.fedora"],
    install_requires=["pyaml", "networkx", "twisted", "autobahn",
                      "python-dateutil", "sqlalchemy==0.7.2",
                      "sqlalchemy-migrate==0.7.2", "Jinja2",
                      "requests", "buildbot"],
)
