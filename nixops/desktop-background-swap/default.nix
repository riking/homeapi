{ lib, pkgs, ... }:

let scriptPackage = pkgs.writeShellApplication {
  name = "desktop-background-swap";
  runtimeInputs = [ pkgs.dbus ];
  text = ''
    BASE_DIR=/home/kane/Videos/Screensaver
    IMAGE_PATH="$(find "$BASE_DIR" -type f | shuf -n1)"

    ${pkgs.dbus}/bin/dbus-send --session --dest=org.kde.plasmashell --type=method_call /PlasmaShell org.kde.PlasmaShell.evaluateScript "string:
    var Desktops = desktops();
    for (i=0;i<Desktops.length; i++) {
      d = Desktops[i];
      d.wallpaperPlugin = 'org.kde.image';
      d.currentConfigGroup = Array('Wallpaper', 'org.kde.image', 'General');
      d.writeConfig('Image', 'file://$IMAGE_PATH');
    }"
  '';
};
in {
  systemd.user.services.desktop-swap = {
    serviceConfig = {
      Type = "simple";
      ExecStart = "${pkgs.runtimeShell} ${scriptPackage}/bin/desktop-background-swap";
    };
  };
  systemd.user.timers.desktop-swap = {
    enable = true;
    description = "Desktop background randomizer";
    timerConfig = {
      OnCalendar = "hourly";
    };
    wantedBy = [ "timers.target" ];
  };
}
