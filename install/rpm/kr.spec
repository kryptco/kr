Name:    kr
Version: %{version}
Release: 1%{?dist}
Url: https://krypt.co
Summary: Krypton daemon -- use an SSH key stored in the Krypton mobile app.
Requires: openssl which
Source: kr.tar.gz
License: All rights reserved.

%description
Krypton daemon -- use an SSH key stored in the Krypton mobile app.

%prep
%autosetup -n src

# to resolve: "ERROR: No build ID note found"
%undefine _missing_build_ids_terminate_build

%build
cd github.com/kryptco/kr
CC=gcc CFLAGS="$RPM_OPT_FLAGS" GOPATH=$PWD/../../../.. make

%install
cd github.com/kryptco/kr
mkdir -p %{buildroot}%{_bindir}/
cp bin/{kr,krd,krssh,krgpg} %{buildroot}%{_bindir}/
mkdir -p %{buildroot}/usr/lib/

%check
cd github.com/kryptco/kr
GOPATH=$PWD/../../../.. make check

%clean
rm -rf %{buildroot}

%files
%{_bindir}/*
%{_libdir}/../lib/*

%post
sudo su ${SUDO_USER:-$USER} <<EOF
mkdir -m 700 -p ~/.ssh
touch ~/.ssh/config
chmod 600 ~/.ssh/config
killall krd 1>/dev/null 2>/dev/null
EOF
echo Krypton is now installed! Type \"kr pair\" to pair with the Krypton app.

%postun

%changelog
