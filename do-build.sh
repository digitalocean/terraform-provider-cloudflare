#!/usr/bin/env bash

# we only run on a linux docker container
# we pretend to be v2.19.2 to make terraform happy with our fork

GOOS=linux GOARCH=amd64 go build \
  -ldflags="-X github.com/cloudflare/terraform-provider-cloudflare/version.ProviderVersion=v2.19.2" \
  -o terraform-provider-cloudflare main.go

shasum -a 256 terraform-provider-cloudflare  | awk '{ print $1 }' > terraform-provider-cloudflare.sha256.txt
