#!/bin/bash

sudo apt-get update
sudo apt-get -y upgrade

#https://golang.org/dl/
wget https://dl.google.com/go/go1.12.9.linux-amd64.tar.gz
sudo tar -xvf go1.12.9.linux-amd64.tar.gz
sudo mv go /usr/local

# Set up environment
export GOROOT=/usr/local/go
export GOPATH=$HOME/ContainerFS/imgGen

export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
