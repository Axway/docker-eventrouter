Name:           qlt-router
Version:        1.0.0
Release:        1
Summary:        QLT Router
BuildArch:      x86_64

License:        Axway License
Source0:        %{name}-%{version}.tar.gz

#Requires:       bash
AutoReq:        no

Prefix:         /

%description
qlt-router filters/routes/transforms/aggregates Axway Sentinel events toward multiple targets : sentinel, kafka, eleasticsearch, files, postgres,... 

%prep
%setup -q


%install
rm -rf $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT/%{_bindir}
cp %{name} $RPM_BUILD_ROOT/%{_bindir}
cp %{name}d $RPM_BUILD_ROOT/%{_bindir}

mkdir -p $RPM_BUILD_ROOT/usr/lib/%{name}
cp -rf lib/* $RPM_BUILD_ROOT/usr/lib/%{name}

%clean
rm -rf $RPM_BUILD_ROOT

%files
%{_bindir}/%{name}
%{_bindir}/%{name}d
%dir /usr/lib/%{name}/
/usr/lib/%{name}/*
