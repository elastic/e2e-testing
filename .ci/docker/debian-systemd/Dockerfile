# This file is part of docker-debian-systemd.
#
# Copyright (c)
#   2018-2019 Alexander Haase <ahaase@alexhaase.de>
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

# This image bases on the regular Debian image. By default the 'latest' tag
# (pointing to the current stable release) of the parent image will be used.
# However, an alternate parent tag may be set by defining the 'TAG' build
# argument to a specific Debian release, e.g. 'stretch' or 'buster'.
ARG  TAG=latest
FROM debian:${TAG}
LABEL maintainer="manuel.delapena@elastic.co"

# Configure the debconf frontend.
#
# This image doesn't include whiptail, dialog, nor the readline perl module.
# Therefore, the debconf frontend will be set to 'teletype' to avoid error
# messages about no dialog frontend could be found.
RUN echo 'debconf debconf/frontend select teletype' | debconf-set-selections


# Install the necessary packages.
#
# In addition to the regular Debian base image, a BASIC set of packages from the
# Debian minimal configuration will be installed. After all packages have been
# installed, the apt caches and some log files will be removed to minimize the
# image.
#
# NOTE: An upgrade will be performed to include updates and security fixes of
#       installed packages that received updates in the Debian repository after
#       the upstream image has been created.
#
# NOTE: No syslog daemon will be installed, as systemd's journald should fit
#       most needs. Please file an issue if you think this should be changed.
RUN apt-get update
RUN apt-get dist-upgrade -y
RUN apt-get install -y --no-install-recommends \
        systemd      \
        systemd-sysv \
        cron         \
        procps       \
        anacron

RUN apt-get clean
RUN rm -rf                        \
    /var/lib/apt/lists/*          \
    /var/log/alternatives.log     \
    /var/log/apt/history.log      \
    /var/log/apt/term.log         \
    /var/log/dpkg.log


# Configure systemd.
#
# For running systemd inside a Docker container, some additional tweaks are
# required. For a detailed list see:
#
# https://developers.redhat.com/blog/2016/09/13/ \
#   running-systemd-in-a-non-privileged-container/
#
# Additional tweaks will be applied in the final image below.

# To avoid ugly warnings when running this image on a host running systemd, the
# following units will be masked.
#
# NOTE: This will not remove ALL warnings in all Debian releases, but seems to
#       work for stretch.
RUN systemctl mask --   \
    dev-hugepages.mount \
    sys-fs-fuse-connections.mount

# The machine-id should be generated when creating the container. This will be
# done automatically if the file is not present, so let's delete it.
RUN rm -f           \
    /etc/machine-id \
    /var/lib/dbus/machine-id




# Build the final image.
#
# To get a minimal image without deleted files in intermediate layers, the
# contents of the image previously built will be copied into a second version of
# the parent image.
#
# NOTE: This method requires buildkit, as the differ of buildkit will copy
#       changed files only and we'll get a minimal image with just the changed
#       files in a single new layer.
#
# NOTE: All settings related to the image's environment (e.g. CMD, ENV and
#       VOLUME settings) need to be set in the following image definition to be
#       used by child images and containers.

FROM debian:${TAG}
COPY --from=0 / /


# Configure systemd.
#
# For running systemd inside a Docker container, some additional tweaks are
# required. Some of them have already been applied above.
#
# The 'container' environment variable tells systemd that it's running inside a
# Docker container environment.
ENV container docker

# A different stop signal is required, so systemd will initiate a shutdown when
# running 'docker stop <container>'.
STOPSIGNAL SIGRTMIN+3

# The host's cgroup filesystem need's to be mounted (read-only) in the
# container. '/run', '/run/lock' and '/tmp' need to be tmpfs filesystems when
# running the container without 'CAP_SYS_ADMIN'.
#
# NOTE: For running Debian stretch, 'CAP_SYS_ADMIN' still needs to be added, as
#       stretch's version of systemd is not recent enough. Buster will run just
#       fine without 'CAP_SYS_ADMIN'.
VOLUME [ "/sys/fs/cgroup", "/run", "/run/lock", "/tmp" ]

# As this image should run systemd, the default command will be changed to start
# the init system. CMD will be preferred in favor of ENTRYPOINT, so one may
# override it when creating the container to e.g. to run a bash console instead.
CMD [ "/sbin/init" ]
