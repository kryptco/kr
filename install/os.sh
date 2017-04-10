#!/bin/bash
# Check for FreeBSD in the uname output
# If it's not FreeBSD, then we move on!
if [ "$(uname -s)" == 'FreeBSD' ]; then
  OS='freebsd'
# Check for a redhat-release file and see if we can
# tell which Red Hat variant it is
elif [ -f "/etc/redhat-release" ]; then
  RHV=$(egrep -o 'Fedora|CentOS|Red\ Hat|Red.Hat' /etc/redhat-release)
  case $RHV in
    Fedora)  OS='fedora';;
    CentOS)  OS='centos';;
   Red\ Hat)  OS='redhat';;
   Red.Hat)  OS='redhat';;
  esac
# Check for debian_version
elif [ -f "/etc/debian_version" ]; then
  OS='debian'
# Check for arch-release
elif [ -f "/etc/arch-release" ]; then
  OS='arch'
# Check for SuSE-release
elif [ -f "/etc/SuSE-release" ]; then
  OS='suse'
fi
# echo the result
echo "$OS"
