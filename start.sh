#!/bin/bash

cd ./mavlink
python3 main.py & 

cd ../
docker compose up

