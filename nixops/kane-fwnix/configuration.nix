# Edit this configuration file to define what should be installed on
# your system. Help is available in the configuration.nix(5) man page, on
# https://search.nixos.org/options and in the NixOS manual (`nixos-help`).

{ config, lib, pkgs, ... }:

let
  dummyLet = 1;
  srcs = import ./npins;
  #pkgs = import srcs.nixpkgs;
  myBluezConfig = pkgs.stdenv.mkDerivation {
    src = [
      (builtins.toFile "share/wireplumber/bluetooth.lua.d/51-bluez-config.lua" ''
	bluez_monitor.properties = {
	  ["bluez5.enable-sbc-xq"] = true,
	  ["bluez5.enable-msbc"] = true,
	  ["bluez5.enable-hw-volume"] = true,
	  ["bluez5.headset-roles"] = "[ hsp_hs hsp_ag hfp_hf hfp_ag ]"
	}
      '')
    ];
  };
  kernel610Pkgs = (import srcs.nixpkgs-kernel-6-10 {});
in {
  imports =
    [
      "${srcs.nixos-hardware}/framework/13-inch/7040-amd"
      ./hardware-configuration.nix
      ../desktop-background-swap
      (import ("${srcs.lix-nixos-module}/module.nix") (
        let lix = srcs.lix.outPath;
	in {
	  inherit lix;
	  versionSuffix = "pre${builtins.substring 0 8 lix.lastModifiedDate}-${lix.shortRev}";
	}
      ))
    ];

  nixpkgs.config.allowUnfreePredicate = pkg: builtins.elem (lib.getName pkg) [
    "discord"
    "steam"
    "steam-original"
    "steam-unwrapped"
  ];

  # Suspend/wake workaround
  hardware.framework.amd-7040.preventWakeOnAC = true;

  # Use the systemd-boot EFI boot loader.
  boot.loader.systemd-boot.enable = true;
  boot.loader.efi.canTouchEfiVariables = true;
  #boot.loader.grub.efiSupport = true;
  #boot.loader.grub.extraEntries = ''
  #  menuentry "Ubuntu" {
  #    search --set=ubuntu --fs-uuid 88171e08-de38-4d46-a390-2c03818f6982
  #    configfile "($ubuntu)/boot/grub/grub.cfg"
  #  }
  #'';

  boot.kernelPackages = kernel610Pkgs.linuxPackages_latest;

  systemd.services.zswap = {
    description = "Enable zswap, set to zstd and Z3FOLD";
    enable = true;
    wantedBy = ["basic.target"];
    path = [ pkgs.bash ];
    serviceConfig = {
      ExecStart = ''${pkgs.bash}/bin/bash -c 'cd /sys/module/zswap/parameters && \
        echo zstd > compressor && echo z3fold > zpool && echo 20 > max_pool_percent && \
        echo 1 > enabled'
        '';
      Type = "oneshot";
    };
  };

  networking.hostName = "kane-fwnix"; # Define your hostname.
  # Pick only one of the below networking options.
  # networking.wireless.enable = true;  # Enables wireless support via wpa_supplicant.
  networking.networkmanager.enable = true;  # Easiest to use and most distros use this by default.

  # Set your time zone.
  time.timeZone = "America/Los_Angeles";

  # Configure network proxy if necessary
  # networking.proxy.default = "http://user:password@proxy:port/";
  # networking.proxy.noProxy = "127.0.0.1,localhost,internal.domain";

  # Select internationalisation properties.
  i18n.defaultLocale = "en_US.UTF-8";
  # console = {
  #   font = "Lat2-Terminus16";
  #   keyMap = "us";
  #   useXkbConfig = true; # use xkb.options in tty.
  # };

  # Fingerprint
  services.fprintd.enable = true;
  # Firmware update
  services.fwupd.enable = true;

  # KDE
  services.xserver.enable = true;
  services.displayManager.sddm.enable = true;
  services.displayManager.defaultSession = "plasma";
  services.desktopManager.plasma6.enable = true;
  programs.dconf.enable = true;
  environment.plasma6.excludePackages = with pkgs.libsForQt5; [
    gwenview
    oxygen
    khelpcenter
    plasma-browser-integration
    print-manager
  ];

  services.tailscale.enable = true;

  # Configure keymap in X11
  services.xserver.xkb.layout = "us";
  # services.xserver.xkb.options = "eurosign:e,caps:escape";

  # Enable CUPS to print documents.
  services.printing.enable = true;

  # Enable sound.
  security.rtkit.enable = true;
  services.pipewire = {
    enable = true;
    alsa.enable = true;
    alsa.support32Bit = true;
    pulse.enable = true;
  };
  hardware.bluetooth.enable = true;
  hardware.bluetooth.powerOnBoot = true;

  # TODO reenable
  #services.pipewire.wireplumber.configPackages."51-bluez-config.lua" = myBluezConfig;

  # Enable touchpad support (enabled default in most desktopManager).
  # services.xserver.libinput.enable = true;

  # Define a user account. Don't forget to set a password with ‘passwd’.
  users.users.kane = {
    isNormalUser = true;
    extraGroups = [ "wheel" ]; # Enable ‘sudo’ for the user.
    packages = with pkgs; [
      firefox
      tree
    ];
  };

  # List packages installed in system profile. To search, run:
  # $ nix search wget
  environment.systemPackages = with pkgs; [
    discord
    fish
    gedit
    git
    krita
    neovim
    npins
    mpv
    prismlauncher
    python313
    stdenv
    (steam.override {
      # Fix for https://forums.factorio.com/113202
      extraProfile = ''
        export XCURSOR_PATH="$(readlink -f /run/current-system/sw)/share/icons"
      '';
    })
    treesheets
    ripgrep
    rustup
    #wayclip
    wl-clipboard-rs
    wget
    (pkgs.callPackage ./vi.nix {})
  ];

  # Some programs need SUID wrappers, can be configured further or are
  # started in user sessions.
  # programs.mtr.enable = true;
  # programs.gnupg.agent = {
  #   enable = true;
  #   enableSSHSupport = true;
  # };

  # List services that you want to enable:

  # Enable the OpenSSH daemon.
  # services.openssh.enable = true;
  programs.ssh.extraConfig = ''
Host khome
  Hostname home.tailscale.riking.org
  Port 59675
  User kane

Host khome-d
  Hostname home.riking.org
  Port 59675
  User kane

Host mchome
  Hostname home.riking.org
  Port 59675
  User mcserver

Host autodelete
  HostName 167.99.61.153
  User autodelete

Host autodelete2
  HostName 159.65.178.37
  HostName 2604:a880:800:a1::1323:2001
  User autodelete

Host autodelete3
  HostName 104.131.62.140
  HostName 2604:a880:800:14::3b:8000
  User autodelete

Host autodelete4
  HostName 64.225.50.158
  HostName 2604:a880:800:10::8f0:7001
  User autodelete

Host autodelete-hetzner1
  HostName 100.93.104.118
  User autodelete

Host whitby
  User riking
  HostName whitby.tvl.fyi

Host whitby.tvl.fyi
  User riking

Host mc-hetzner
  HostName 5.78.79.133
  User mcserver
  '';

  # Open ports in the firewall.
  # networking.firewall.allowedTCPPorts = [ ... ];
  # networking.firewall.allowedUDPPorts = [ ... ];
  # Or disable the firewall altogether.
  # networking.firewall.enable = false;

  nix.settings = {
    experimental-features = "nix-command flakes";
  };
  # FIXME(v24.05): change following two rules to
  nixpkgs.flake.source = srcs.nixpkgs;
  nix.registry.nixpkgs.to = {
    type = "path";
    path = srcs.nixpkgs;
  };
  nix.nixPath = ["nixpkgs=flake:nixpkgs"];

  # Copy the NixOS configuration file and link it from the resulting system
  # (/run/current-system/configuration.nix). This is useful in case you
  # accidentally delete configuration.nix.
  system.copySystemConfiguration = true;

  # This option defines the first version of NixOS you have installed on this particular machine,
  # and is used to maintain compatibility with application data (e.g. databases) created on older NixOS versions.
  #
  # Most users should NEVER change this value after the initial install, for any reason,
  # even if you've upgraded your system to a new NixOS release.
  #
  # This value does NOT affect the Nixpkgs version your packages and OS are pulled from,
  # so changing it will NOT upgrade your system.
  #
  # This value being lower than the current NixOS release does NOT mean your system is
  # out of date, out of support, or vulnerable.
  #
  # Do NOT change this value unless you have manually inspected all the changes it would make to your configuration,
  # and migrated your data accordingly.
  #
  # For more information, see `man configuration.nix` or https://nixos.org/manual/nixos/stable/options#opt-system.stateVersion .
  system.stateVersion = "23.11"; # Did you read the comment?

}

