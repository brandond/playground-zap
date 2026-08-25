#!/usr/bin/bash -x

docker buildx build --output type=image,push=true,name=brandond/playground-zap:latest .
