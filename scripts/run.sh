#!/usr/bin/env bash

scripts/build.sh && docker run --rm --name=compiler -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 gcr.io/${PROJECT}/compiler