%define _build_id_links none

Name:           axia
Version:        %{version}
Release:        %{release}
Summary:        The Axia platform binaries
URL:            https://github.com/sankar-boro/%{name}
License:        BSD-3
AutoReqProv:    no

%description
Axia is an incredibly lightweight protocol, so the minimum computer requirements are quite modest.

%files
/usr/local/bin/axia
/usr/local/lib/axia
/usr/local/lib/axia/evm

%changelog
* Mon Oct 26 2020 Charlie Wyse <charlie@avalabs.org>
- First creation of package

