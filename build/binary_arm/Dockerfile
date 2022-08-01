FROM multiarch/ubuntu-debootstrap:arm64-bionic
LABEL maintainer="Mingxia Cai"

# install gcc make git
RUN apt-get update \
    && apt-get install -y build-essential git

# The image is used to build chaosblade for musl
RUN wget http://www.musl-libc.org/releases/musl-1.1.21.tar.gz \
    && tar -zxvf musl-1.1.21.tar.gz \
    && rm musl-1.1.21.tar.gz \
    && cd musl* \
    && ./configure \
    && make \
    && make install \
    && rm -rf musl*

# install go
RUN wget https://dl.google.com/go/go1.13.10.linux-arm64.tar.gz \
    && tar -C /usr/local -xzf go1.13.10.linux-arm64.tar.gz \
    && rm -rf go1.13.10.linux-arm64.tar.gz

ENV CC /usr/local/musl/bin/musl-gcc
ENV GOOS linux
ENV PATH /usr/local/go/bin:$PATH


ENTRYPOINT [ "make" ]
