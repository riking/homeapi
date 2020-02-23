#!/bin/bash

eval "$(/tank/tljh/home/anaconda3/bin/conda shell.bash hook)"

jupyter notebook --config="$HOME"/.jupyter/jupyter_notebook_config.py
