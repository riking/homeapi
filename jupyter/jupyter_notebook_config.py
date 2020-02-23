# ...
# ...

c.NotebookApp.allow_remote_access = True

c.NotebookApp.ip = 'localhost'
c.NotebookApp.port = 2206

c.NotebookApp.base_url = '/ipython'

# unclear how to secure this away from the notebook user?
c.NotebookApp.cookie_secret_file = '/tank/tljh/home/.jupyter/cookie_secret.txt'

c.NotebookApp.custom_display_url = 'https://admin.riking.org/ipython/'

c.NotebookApp.notebook_dir = '/tank/tljh/home/notebooks/'

# ...
# ...
