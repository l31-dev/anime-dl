# syntax=docker/dockerfile:1

################################################################################
# Pick a base image to serve as the foundation for the other build stages in
# this file.
# Using the latest tag for the debian:stable-slim image.
FROM debian:stable-slim AS base

################################################################################
# Create a stage for building/compiling the application.
# Copy the current directory to /app/ and make gazes-cli executable.
FROM base AS build
WORKDIR /app
COPY . /app/
RUN apt update && apt install -y curl jq vlc fzf gpg ffmpeg dos2unix && \
    dos2unix /app/gazes-cli && \
    chmod +x /app/gazes-cli && \
    mkdir -p /etc/apt/keyrings && \
    curl -fsSL https://repo.charm.sh/apt/gpg.key | gpg --dearmor -o /etc/apt/keyrings/charm.gpg && \
    echo "deb [signed-by=/etc/apt/keyrings/charm.gpg] https://repo.charm.sh/apt/ * *" | tee /etc/apt/sources.list.d/charm.list && \
    apt update && apt install -y gum


# Create a non-privileged user that the app will run under.
ARG UID=10001
USER root

ENTRYPOINT [ "bash", "/app/gazes-cli" ]
