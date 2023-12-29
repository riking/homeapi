{ lib, pkgs, ... }:

pkgs.stdenv.mkDerivation {
  name = "vi";
  phases = [ "installPhase" ];
  nativeBuildInputs = [ pkgs.neovim ];
  installPhase = ''
    mkdir -p $out/bin
    ln -s ${pkgs.neovim}/bin/nvim $out/bin/vi
    ln -s ${pkgs.neovim}/bin/nvim $out/bin/vim
  '';
}
