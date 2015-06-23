Builder
=======

# Local configuration

Private information such as GitHub authentication credentials are loaded
from the ``~/.config/hawaii-builder.ini`` configuration file.

You must create one on the master before running it.

Here's the format:

```sh
[Buildbot]
URL=<Master URL ending with />

[GitHub]
ClientId=<GitHub authentication client ID>
ClientSecret=<GitHub authentication client secret>
```

# Build master server setup

On the build master server, install buildbot:

```sh
sudo pacman -S base-devel git python2-pip python2-virtualenv

mkdir ~/buildbot
cd ~/buildbot

virtualenv-2.7 --no-site-packages env
source env/bin/activate

pip install --upgrade pip
pip install mock pyaml networkx twisted autobahn python-dateutil sqlalchemy==0.7.2 sqlalchemy-migrate==0.7.2 Jinja2 requests

git clone --depth 1 https://github.com/buildbot/buildbot src
cd src

pushd master
python setup.py install
popd

pushd pkg
python setup.py install
popd

make prebuilt_frontend
```

Then create the master configuration:

```sh
cd ~/buildbot
buildbot create-master master
```

From the clone of this repository:

```sh
cp master.cfg ~/buildbot/master/master.cfg
```

# Slave setup

On the build slave, install needed packages:

```sh
sudo pacman -S devtools base-devel git python2-pip python2-virtualenv

mkdir ~/buildbot
cd ~/buildbot

virtualenv-2.7 --no-site-packages env
source env/bin/activate

pip install --upgrade pip
pip install mock pyaml networkx twisted

git clone https://github.com/buildbot/buildbot src
cd src

pushd slave
python setup.py install
popd
```

Also install (clean-chroot-manager)[https://bbs.archlinux.org/viewtopic.php?id=168421].

Create the slaves:

```sh
cd ~/buildbot
mkdir -p slaves
buildslave create-slave slaves/slave1 localhost:9989 slave1 password
buildslave create-slave slaves/slave2 localhost:9989 slave2 password
echo "i686 host" > slaves/slave1/info/host
echo "x86_64 host" > slaves/slave2/info/host
echo "Full Name <email@address>" > slaves/slave1/info/admin
echo "Full Name <email@address>" > slaves/slave2/info/admin
```
