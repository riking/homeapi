
```bash
root$ zfs create tank/tljh
root$ adduser notebook --homedir=/tank/tljh/home
root$ mv start.sh /tank/tljh/start.sh; chown root:root /tank/tljh/start.sh

root$ su notebook
notebook$ wget ....Anaconda3-2019.10-Linux-x86_64.sh
notebook$ # (run Anaconda setup)
```
