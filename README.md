Nikita
======

# Build master server setup

On the build master server, install buildbot:

```sh
sudo pacman -S base-devel git python2-pip python2-virtualenv npm nodejs

mkdir ~/buildbot
cd ~/buildbot

virtualenv2 --no-site-packages buildbotenv
source buildbotenv/bin/activate

pip install --upgrade pip
pip install mock pyaml networkx

git clone https://github.com/buildbot/buildbot buildbotsrc

cd buildbotsrc/master
python setup.py install

cd ../slave
python setup.py install

cd ../pkg
python setup.py install

cd ..
make prebuilt_frontend

cd ../www
for i in base codeparameters console_view md_base waterfall_view; do
pushd $i
python setup.py install
popd
done
```

Then create the master configuration:

```sh
cd ~/buildbot
mkdir -p masters
cd masters
buildbot create-master archlinux
```

From the clone of this repository:

```sh
cp archlinux.cfg ~/buildbot/masters/archlinux/master.cfg
```

# Slave setup

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
