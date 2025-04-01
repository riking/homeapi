#!/usr/bin/env bash

cd $(dirname $0)
cmd=${1:-switch}
shift

nixpkgs_pin=$(nix eval --raw -f npins/default.nix nixpkgs)
export NIX_PATH="nixpkgs=${nixpkgs_pin}:nixos-config=${PWD}/configuration.nix"

# without --fast, nixos-rebuild will compile nix and use the compiled nix to evaluate the config
# nom is nix-output-monitor
sudo /usr/bin/env NIX_PATH="${NIX_PATH}" nixos-rebuild "$cmd" --fast "$@" |& nom
