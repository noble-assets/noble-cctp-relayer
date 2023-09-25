FROM --platform=$BUILDPLATFORM golang:1.20-alpine AS build-env

RUN apk add --update --no-cache openssh curl make git libc-dev bash gcc linux-headers eudev-dev

ARG TARGETARCH
ARG BUILDARCH

RUN if [ "${TARGETARCH}" = "arm64" ] && [ "${BUILDARCH}" != "arm64" ]; then \
        wget -c https://musl.cc/aarch64-linux-musl-cross.tgz -O - | tar -xzvv --strip-components 1 -C /usr; \
    elif [ "${TARGETARCH}" = "amd64" ] && [ "${BUILDARCH}" != "amd64" ]; then \
        wget -c https://musl.cc/x86_64-linux-musl-cross.tgz -O - | tar -xzvv --strip-components 1 -C /usr; \
    fi

WORKDIR /src

# Only necessary while github.com/circlefin/noble-cctp is private
ARG GITAUTH

RUN mkdir -p /root/.ssh; \
    echo "$GITAUTH" | base64 -d > /root/.ssh/id_ed25519; \
    chmod 600 /root/.ssh/id_ed25519; \
    git config --global --add url."git@github.com:circlefin/noble-cctp.git".insteadOf "https://github.com/circlefin/noble-cctp"; \
    ssh-keyscan github.com >> /root/.ssh/known_hosts

ADD . .

RUN if [ "${TARGETARCH}" = "arm64" ] && [ "${BUILDARCH}" != "arm64" ]; then \
        export CC=aarch64-linux-musl-gcc CXX=aarch64-linux-musl-g++;\
    elif [ "${TARGETARCH}" = "amd64" ] && [ "${BUILDARCH}" != "amd64" ]; then \
        export CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++; \
    fi; \
    GOPRIVATE='github.com/circlefin/noble-cctp'; \
    GOOS=linux GOARCH=$TARGETARCH CGO_ENABLED=1; \
    LDFLAGS='-linkmode external -extldflags "-static"'; \
    go install -ldflags "-s -w $LDFLAGS"

RUN if [ -d "/go/bin/linux_${TARGETARCH}" ]; then mv /go/bin/linux_${TARGETARCH}/* /go/bin/; fi

# Use minimal busybox from infra-toolkit image for final scratch image
FROM ghcr.io/strangelove-ventures/infra-toolkit:v0.0.8 AS busybox-min
RUN addgroup --gid 1000 -S strangelove && adduser --uid 100 -S strangelove -G strangelove

# Use ln and rm from full featured busybox for assembling final image
FROM busybox:1.34.1-musl AS busybox-full

# Build final image from scratch
FROM scratch

LABEL org.opencontainers.image.source="https://github.com/strangelove-ventures/noble-cctp-relayer"

WORKDIR /bin

# Install ln (for making hard links) and rm (for cleanup) from full busybox image (will be deleted, only needed for image assembly)
COPY --from=busybox-full /bin/ln /bin/rm ./

# Install minimal busybox image as shell binary (will create hardlinks for the rest of the binaries to this data)
COPY --from=busybox-min /busybox/busybox /bin/sh

# Add hard links for read-only utils
# Will then only have one copy of the busybox minimal binary file with all utils pointing to the same underlying inode
RUN for b in \
  cat \
  date \
  df \
  du \
  env \
  grep \
  head \
  less \
  ls \
  md5sum \
  nc \
  nslookup \
  ping \
  ping6 \
  pwd \
  sha1sum \
  sha256sum \
  sha3sum \
  sha512sum \
  sleep \
  stty \
  tail \
  tar \
  tee \
  tr \
  watch \
  which \
  ; do ln sh $b; done

#  Remove write utils
RUN rm ln rm

# Install chain binaries
COPY --from=build-env /go/bin/noble-cctp-relayer /bin

# Install trusted CA certificates
COPY --from=busybox-min /etc/ssl/cert.pem /etc/ssl/cert.pem

# Install strangelove user
COPY --from=busybox-min /etc/passwd /etc/passwd
COPY --from=busybox-min --chown=100:1000 /home/strangelove /home/strangelove

WORKDIR /home/strangelove
USER strangelove

ENTRYPOINT ["noble-cctp-relayer", "start", "--config", "./config/testnet.yaml"]