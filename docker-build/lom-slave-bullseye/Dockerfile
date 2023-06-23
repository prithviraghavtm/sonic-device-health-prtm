FROM debian:bullseye

MAINTAINER remanava@microsoft.com

COPY ["no-check-valid-until", "/etc/apt/apt.conf.d/"]
COPY ["apt-multiple-retries", "/etc/apt/apt.conf.d/"]
COPY ["sources.list.amd64", "/etc/apt/sources.list"]
COPY ["rsyslog.conf", "/etc/"]
COPY ["lom_start.sh", "/usr/local/bin/"]

## Make apt-get non-interactive
ENV DEBIAN_FRONTEND=noninteractive

# Auto-Generated for buildinfo
ARG SONIC_VERSION_CACHE
ARG SONIC_VERSION_CONTROL_COMPONENTS
COPY ["buildinfo", "/usr/local/share/buildinfo"]
# Remove unused
COPY vcache/ /sonic/target/vcache/lom-slave-bullseye
RUN dpkg -i /usr/local/share/buildinfo/sonic-build-hooks_1.0_all.deb
ENV IMAGENAME=lom-slave-bullseye
ENV DISTRO=bullseye
RUN pre_run_buildinfo lom-slave-bullseye

RUN apt-get update && apt-get install -y \
        apt-utils \
        default-jre-headless \
        openssh-server \
        curl \
        wget \
        unzip \
        gzip \
        git \
        build-essential \
        libtool \
        lintian \
        sudo \
        dh-make \
        dh-exec \
        kmod \
        libtinyxml2-dev \
        python-all \
        python-dev \
        python-setuptools \
        python3 \
        python3-pip \
        libncurses5-dev \
        texinfo \
        dh-autoreconf \
        doxygen \
        devscripts \
        git-buildpackage \
        perl-modules \
        libclass-accessor-perl \
        libswitch-perl \
        libzmq5 \
        libzmq3-dev \
        uuid-dev \
        uuid-runtime \
        jq \
        cron \
# For quagga build
        libreadline-dev \
        texlive-latex-base \
        texlive-plain-generic \
        texlive-fonts-recommended \
        libpam0g-dev \
        libpam-dev \
        libcap-dev \
        imagemagick \
        ghostscript \
        groff \
        libpcre3-dev \
        gawk \
        chrpath \
# For frr build
        libc-ares-dev \
        libsnmp-dev \
        libjson-c-dev \
        libsystemd-dev \
        python3-ipaddr \
        libcmocka-dev \
#
        python3-all-dev \
        python3-all-dbg \
#
        install-info \
        logrotate \
# For libnl3 (local) build
        cdbs \
# For SAI meta build
        libxml-simple-perl \
        graphviz \
        aspell \
# For SAI meta rpc build - make rpc
        libgetopt-long-descriptive-perl \
        libconst-fast-perl \
        libtemplate-perl \
        libnamespace-autoclean-perl \
        libmoose-perl \
        libmoosex-aliases-perl \
# For linux build
        bc \
        fakeroot \
        build-essential \
        devscripts \
        quilt \
        stgit \
        sbsigntool \
# For platform-modules build
        module-assistant \
# For thrift build\
        gem2deb \
        libevent-dev \
        libglib2.0-dev \
#
        python3-all-dev \
#
        python3-twisted \
        phpunit \
        libbit-vector-perl \
        openjdk-11-jdk \
        javahelper \
        maven-debian-helper \
        ant \
        libhttpclient-java \
        libslf4j-java \
        libservlet3.1-java \
        pkg-php-tools \
# For mellanox sdk build
        libpcre3 \
        libpcre3-dev \
        byacc \
        flex \
        libglib2.0-dev \
        bison \
        expat \
        libexpat1-dev \
        dpatch \
        libdb-dev \
        libiptc-dev \
        libxtables-dev \
# For mellanox sai build
        libtool-bin \
        libxml2-dev \
# For BFN sdk build
        libusb-1.0-0-dev \
        libcurl4-openssl-dev \
        libunwind8-dev \
        telnet \
        libc-ares2 \
        libgoogle-perftools4 \
# For build image
        cpio \
        squashfs-tools \
        zip \
# For broadcom sdk build
        linux-compiler-gcc-10-x86 \
        linux-kbuild-5.10 \
# teamd build
        libdaemon-dev \
        libdbus-1-dev \
        libjansson-dev \
# For cavium sdk build
        libpcap-dev \
        dnsutils \
        libusb-dev \
# For cisco sdk build
        libgrpc++1 \
        libabsl-dev \
# For debian image reconfiguration
        augeas-tools \
# For p4 build
        libyaml-dev \
        libevent-dev \
        libjudy-dev \
        libedit-dev \
        libnanomsg-dev \
        python3-stdeb \
# For redis build
        libjemalloc-dev \
        liblua5.1-0-dev \
        lua-bitop-dev  \
        lua-cjson-dev \
# For mft kernel module build
        dkms \
# For Jenkins static analysis, unit testing and code coverage
        cppcheck \
        clang \
        pylint \
        python3-pytest \
        python3-venv \
        gcovr \
        python3-pytest-cov \
        python3-pytest-cov \
        python3-parse \
# For snmpd
        default-libmysqlclient-dev \
        libssl-dev \
        libperl-dev \
        libpci-dev \
        libpci3 \
        libsensors5 \
        libsensors4-dev \
        libwrap0-dev \
# For lldpd
	debhelper \
        autotools-dev \
        libbsd-dev \
        pkg-config \
        check \
# For mpdecimal
        docutils-common \
        libjs-sphinxdoc \
        libjs-underscore \
        python3-docutils \
        python3-jinja2 \
        python3-markupsafe \
        python3-pygments \
        python3-roman \
        python3-sphinx \
        sphinx-common \
        python3-sphinx \
# For sonic config engine testing
        python3-dev \
# For lockfile
        procmail \
# For gtest
        libgtest-dev \
        cmake \
# For gmock
        libgmock-dev \
# For pam_tacplus build
        autoconf-archive \
# For iproute2
        cm-super-minimal \
        libatm1-dev \
        libbpf-dev \
        libelf-dev \
        libmnl-dev \
        libselinux1-dev \
        linuxdoc-tools \
        lynx \
        texlive-latex-extra \
        texlive-latex-recommended \
        iproute2 \
# For bash
        texi2html \
        sharutils \
        locales \
        time \
        man2html-base \
        libcunit1 \
        libcunit1-dev \
# For initramfs
        shellcheck \
        bash-completion \
# For sonic vs image build
        dosfstools \
        qemu-kvm \
        libvirt-clients \
# For ntp
        autogen \
        libopts25-dev \
        pps-tools \
        dh-apparmor \
# For lm-sensors
        librrd8 \
        librrd-dev \
        rrdtool \
# For kdump-tools
        liblzo2-dev \
# For iptables
        libnetfilter-conntrack-dev \
        libnftnl-dev \
# For SAI3.7
        protobuf-compiler \
        libprotobuf-dev \
        xxd \
# For DHCP Monitor tool
        libexplain-dev \
        libevent-dev \
# For libyang
        swig \
# For build dtb
        device-tree-compiler \
# For sonic-mgmt-framework
        autoconf \
        m4 \
        libxml2-utils \
        xsltproc \
        python3-lxml \
        libexpat1-dev \
        libcurl4 \
        libcjson-dev \
# For WPA supplication
        qtbase5-dev          \
        aspell-en            \
        libssl-dev           \
        dbus                 \
        libdbus-1-dev        \
        libdbus-glib-1-2     \
        libdbus-glib-1-dev   \
        libreadline-dev      \
        libncurses5-dev      \
        libpcsclite-dev      \
        docbook-to-man       \
        docbook-utils        \
# For kdump-tools
        libbz2-dev \
# For linkmgrd
        libboost-dev \
        libboost-program-options-dev \
        libboost-system-dev \
        libboost-serialization1.74-dev \
        libboost-thread-dev \
        libboost-atomic-dev \
        libboost-chrono-dev \
        libboost-container-dev \
        libboost-context-dev \
        libboost-contract-dev \
        libboost-coroutine-dev \
        libboost-date-time-dev \
        libboost-fiber-dev \
        libboost-filesystem-dev \
        libboost-graph-parallel-dev \
        libboost-log-dev \
        libboost-regex-dev \
        googletest \
        libgtest-dev \
        libgmock-dev \
        libgcc-10-dev \
# For sonic-host-services build
        libcairo2-dev \
        libdbus-1-dev \
        libgirepository1.0-dev \
        libsystemd-dev \
        pkg-config \
# For audisp-tacplus
        libauparse-dev \
        auditd

RUN apt-get -y build-dep openssh

# Build fix for ARM64 and ARMHF /etc/debian_version

# Build fix for ARMHF bullseye libsairedis

## Config dpkg
## install the configuration file if it’s currently missing
RUN sudo augtool --autosave "set /files/etc/dpkg/dpkg.cfg/force-confmiss"
## combined with confold: overwrite configuration files that you have not modified
RUN sudo augtool --autosave "set /files/etc/dpkg/dpkg.cfg/force-confdef"
## do not modify the current configuration file, the new version is installed with a .dpkg-dist suffix
RUN sudo augtool --autosave "set /files/etc/dpkg/dpkg.cfg/force-confold"
# For linux build
RUN apt-get -y build-dep linux

# Install golang
RUN wget -O golang-go.tar.gz https://go.dev/dl/go1.20.3.linux-amd64.tar.gz
RUN mkdir -p /usr/local/go1.20.3
RUN tar -C /usr/local/go1.20.3 -xzf golang-go.tar.gz

RUN pip3 install --upgrade pip
RUN apt-get purge -y python3-pip python3-yaml

# For building Python packages
RUN pip3 install setuptools==49.6.00
RUN pip3 install wheel==0.38.1

# For building sonic-utilities
RUN pip3 install fastentrypoints mock

# For running Python unit tests
RUN pip3 install pytest-runner==5.2
RUN pip3 install nose==1.3.7
RUN pip3 install mockredispy==2.9.3

# Fix CVE-2021-23437, need to build and install libjpeg-dev for armhf for pillow 9.4.0
RUN pip3 install pillow==9.4.0

# For p4 build
RUN pip3 install \
         ctypesgen==1.0.2 \
         crc16

# For sonic config engine testing
# Install pyangbind here, outside sonic-config-engine dependencies, as pyangbind causes enum34 to be installed.
# enum34 causes Python 're' package to not work properly as it redefines an incompatible enum.py module
# https://github.com/robshakir/pyangbind/issues/232
RUN pip3 install pyangbind==0.8.1
RUN pip3 uninstall -y enum34

# For templating
RUN pip3 install j2cli==0.3.10

# For sonic-mgmt-framework
RUN pip3 install "PyYAML==5.4.1"
RUN pip3 install "lxml==4.9.1"

# For sonic-platform-common testing
RUN pip3 install redis

# For vs image build
RUN pip3 install pexpect==4.8.0

# For sonic-swss-common testing
RUN pip3 install Pympler==0.8

# For sonic_yang_model build
RUN pip3 install pyang==2.4.0

# For mgmt-framework build
RUN pip3 install mmh3==2.5.1

RUN pip3 install parameterized==0.8.1

RUN apt-get install -y xsltproc

# Install dependencies for isc-dhcp-relay build
RUN apt-get -y build-dep isc-dhcp

# Install vim
RUN apt-get install -y vim

# Install tree
RUN apt-get install -y tree

# Install rsyslog
RUN apt-get install -y rsyslog
RUN cd /usr/src/gtest && cmake . && make -C /usr/src/gtest

RUN mkdir /var/run/sshd
EXPOSE 22

# Install depot-tools (for git-retry)
RUN git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git /usr/share/depot_tools
ENV PATH /usr/share/depot_tools:$PATH

# Install dependencies for dhcp relay test
RUN pip3 install parameterized==0.8.1
RUN pip3 install pyfakefs

# Install docker engine 20.10 inside docker and enable experimental feature
RUN apt-get update
RUN apt-get install -y \
           apt-transport-https \
           ca-certificates \
           curl \
           gnupg2 \
           software-properties-common
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | sudo apt-key add -
RUN add-apt-repository \
           "deb [arch=amd64] https://download.docker.com/linux/debian \
           $(lsb_release -cs) \
           stable"
RUN apt-get update
RUN apt-get install -y docker-ce=5:20.10.14~3-0~debian-bullseye docker-ce-cli=5:20.10.14~3-0~debian-bullseye containerd.io=1.5.11-1
RUN echo "DOCKER_OPTS=\"--experimental --storage-driver=vfs \"" >> /etc/default/docker
RUN update-alternatives --set iptables /usr/sbin/iptables-legacy

# Install m2crypto package, needed by SWI tools
RUN pip3 install m2crypto==0.36.0

# Install swi tools
RUN pip3 install git+https://github.com/aristanetworks/swi-tools.git@bead66bf261770237f7dd21ace3774ba04a017e9

# Install Bazel build system (amd64 and arm64 architectures are supported using this method)
# TODO(PINS): Remove once pre-build Bazel binaries are available for armhf (armv7l)
ARG bazelisk_url=https://github.com/bazelbuild/bazelisk/releases/latest/download/bazelisk-linux-amd64
RUN curl -fsSL -o /usr/local/bin/bazel ${bazelisk_url} && chmod 755 /usr/local/bin/bazel
# Bazel requires "python"
# TODO(PINS): remove when Bazel is okay with "python3" binary name
RUN apt install -y python-is-python3

RUN post_run_buildinfo lom-slave-bullseye 

RUN post_run_cleanup lom-slave-bullseye 

