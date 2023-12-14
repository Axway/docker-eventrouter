# Event router LoadBalancer

`event-router` is a very *experimental* tool to evaluate the viability of dispatching Sentinel events accross several Sentinel frontend/acquisition servers.

ST/CFT/....   --> EventRouter --> event-router --> sentinel1, sentinel2, ...

## Usage

```bash
    ./event-router --qlt_port=3333 --sentinel_addrs=sentinel1:3333, sentinel2:3333,.... --sentinel_connections=1
```

`--sentinel_connections` : is the number of connections per sentinel server. More than one *may* speedup acquisition

## Caveats

- `event-router` may lose messages as acknowlegment are done too quickly (to be fixed)
- if multiple instances of `event-router` are used, they should have the same parameters in particular `--sentinel_addrs` and `--sentinel_connections`
- the first sentinel server will receive all non XFBTransfer messages (cyclelink, xfblogs...) (to be fixed)
- linux only for now


## ER3 vs ER2

- XML config
- DMZ support (QLTSRV / QLTREQ)
```xml
<Target name="SRV" defaultXntf="yes" defaultXml="yes">
<Access mode="QLTSRV"/>
```

```xml
<Target name="REQ" defaultXntf="yes" defaultXml="yes">
<Access mode="QLTREQ" port="1505" addr="hpx3.pa.axway" ident="SRV"/>
<SendIf period ="20"/>
```

- Proxy ? Socks4? tcp_org_port_file / tcp_proxi_file

- ER Entities : DISP, LOG, ZLGR(z/OS), MQIN (WebSpherMQ), MQOUT (WebSphereMQ), NET, NETS, SNTL
- Overflow File, Batch

- Trkagent.ini
```ini
[Agent] #event router name, target parametrs file, logging, message maxsize
name=
target_parameters_file=target.xml #"$trk_home_dir/conf/target.xml"
security_profile_file=sslconf.ini 1"$trk_home_dir/conf/sslconf.ini"
encrypt_key_file= #"$trk_home_dir/conf/crypt.key"
log=0
message_size=4000
#z/OS api_file,api_timer,queue (z/OS)
#z/OS[System] z/OS
[TcpSource]
sap=2305 #port
sapssl=0 #tlsPort
profile=<ini> 
local_adress= #host
incoming_max=10 
tcp_org_port_file=none 
tcp_listen_max_retry=
tcp_watch_deplay=
tcp_listen_delay_retry=
ipv6_disable_connect=0 #deprecated
ipv6_disable_listen=0 #deprecated

[DefaultTarget]
directory=./data #"$trk_home_dir/data"
max_messages=10000
timeout=5
short_wait=10
jump_wait=20
long_wait=300
keep_connection=30 #0 should mean forever?
heartbeat=0 --> see heartbeat messages
address=<sentinel-addr>
port=<sentinel-addr>
backup_address=<sentinel-backup-addr>
backup_port=<sentinel-backup-port>
ssl=No #(Yes,Y,No,N)
profile=<ssl-profile-name>

[Sentinel] SNTL entity? / loggin level
active=0 #/1 SNTL entity activate
log=0 #SNTL entity
#z/OS buffer_size
sap= #QLTSRV port
sapssl=0 #QLTSRV port
profile=<security-section-name>

[MQSeries]

[SSL-PROFILE]
SSL_USER_CERTIFICATE_FILE = $trk_home_dir/conf/user.p12
SSL_USER_CERTIFICATE_FORMAT = PKCS12
SSL_USER_CERTIFICATE_PASSWORD_FILE = $trk_home_dir/conf/user.pw

```

EventRouter.DefaultTarget.Access
    mode = QLT/HTTP/MQSeries/QLTREQ/QLTSRV
    port
    backup_addr
    backup_port
    ssl=yes/no
    profile=
    ident= QLTREQ
```xml
<TrkEventRouterCfg>
    <TrkXml Version="1" />
    <EventRouter name="EventRouterOne">
        <DefaultTarget>
            <Access addr="sentinelserver.pc.pa.sopra" mode="QLT" port="1302"/>
            <Connection short_wait="60" jump_wait="60" long_wait="3600" keep_connection="30" timeout="10"/>
            <File name="" directory="/EventRouter/data/" max_ messages="1000"/>
        </DefaultTarget>
        <Exceptions keep="yes" target="Errmsg"/>
    </EventRouter>
    <Target name="XNTF" defaultXntf="yes" defaultXml="no">
        <Access mode="QLT" addr="SentinelServer" port="1302"/>
        <Connection short_wait="60" jump_wait="120" long_wait="3600" keep_connection="30" timeout="10"/>
        <File name="OverflowOne" directory="/EventRouter/data/" max_messages="1000"/>
        <Batch active="yes" sendAlert="yes">
            <File name="BatchOne" directory="/EventRouter/data/" max_messages="1000"/>
            <SendIf nb_messages="200" period="60"/>
        </Batch>
    </Target>
    <Target name="QLTXML" defaultXntf="no" defaultXml="yes">
        <Access port="1303" mode="QLT"/>    
    </Target>
    <Target name="SCOPEV1" defaultXntf="yes" defaultXml="no">
        <Access addr="RS16.pa.sopra" port="44444" mode="QLT"/>
    </Target>
    <Target name="SENTINEL" defaultXntf="yes" defaultXml="yes">
        <Access addr="hpx11.pa.sopra" port="1305" mode="QLT"/>
    </Target>
    <Target name="Errmsg">
        <Batch active="yes" sendAlert="yes">
            <File name="Errors" directory="/EventRouter/data/data" max_messages="1000"/>
        </Batch>
    </Target>
    <Route object="All" default_Notify="NotifyIf">
        <Condition notify="NotifyIf" target="SENTINEL" if="[PRODUCTNAME]=API-SCRIPT AND [APPLICATION]=AgentTest"></Condition>
    </Route>
    <Route object="XFBLog" default_Notify="NotifyIf">
        <Condition notify="NotifyIf" target="SENTINEL" if="[APPLICATION]=AgentTest AND [SEVERITY]=1"></Condition>
    </Route>
    <Authorization>
        <exeCmd name="SCOPEV1" password="passwdSCOP1"/>
        <exeCmd name="SENTINEL" password="passwdSENT"/>
    </Authorization>
</TrkEventRouterCfg>
```

if syntax
```
<exp> ::= <exp> 'AND' <exp>
        <exp> 'OR' <exp>
        <literal>
literal ::= <a> <op> <a>
a ::= '[' identifier ']' | <identifier> '(' <params> ')' | <identifier> | <string>
params ::= <a> , <params> | <a>
op := '=' | 'NOT' | 'INFEQ' | '>' |  '<' | 'SUPEQ'
identifier := r'[a-zA-Z0-9_]+'
string := r'[]*'
```

```xml
<?xml version="1.0" encoding="ISO-8859-1"?>
<TrkEventRouterCfg>
   <TrkXml version="1.0" />
   <EventRouter name="QUAL">
   </EventRouter>
   <Target name="DEFAULT" defaultXntf="yes" defaultXml="norules">
   </Target>
   <Target name="MODULO2_0" defaultXntf="no" defaultXml="no">
       <Access port="1305" addr="slnxcftgov.lab1.lab.ptx.axway.int"  />
   </Target>
   <Target name="MODULO2_1" defaultXntf="no" defaultXml="no">
       <Access port="1306" addr="slnxcftgov.lab1.lab.ptx.axway.int"  />
   </Target>
   <Route object="ALL" default_Notify="NotifyIf">
     <Condition notify="NotifyIf" target="MODULO2_0" if="[APPLICATION] NOT CONTROL AND HASHMOD([CYCLEID],2,0)"/>
     <Condition notify="NotifyIf" target="MODULO2_1" if="[APPLICATION] NOT CONTROL AND HASHMOD([CYCLEID],2,1)"/>
   </Route>
</TrkEventRouterCfg>
```

## Commands

```bash
agtcmd 
    end,quit,bye,exit
    help
    ?
    display help
about 
start
stop -- stop ER processes
kill -- kill all ER processes
status -- display processes

#entities
    detail -e <entity>
    pause/restart -e <entity> MQIN/ZLGR
    loadfile -e <entity>
# target
    reset -t <target>
    disable -r <target>
    enable -t <target>
    froze -t <targer>
    force -t <target>
    count [-t <target>]
    resetcount [-t target]
    loadconfig 
    purge -t <target> -n <number> [-ft <file-type>]
    display [-t target] #display statuses : ENABLED/DISABLED/ERROR/INIT

# traces
    logarchive
    log -l <level> [-e <entity>]
    trace -l <level> [-e <entity>]

```

## Log Messages

<date> <time> <msg-code> <entity/process> <Level> <msg-label> <msg>

## agtcrypt

```bash
agtcrypt --genkey --pass PASSWORD --keyfname FILENAME --saltfname FILENAME
agtcrypt –encrypt --text P12_PASSWORD --keyfname FILENAME –textfname FILENAME
agtcrypt –decrypt --keyfname FILENAME --text PASSWORDDATA | –textfname FILENAME
agtcrypt --renewkey --pass PASSWORD --keyfname FILENAME --saltfname FILENAME --oldpass PASSWORD
```
