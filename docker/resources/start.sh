#!/bin/bash
#
# DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS HEADER.
#
# Copyright (c) 2019 Axway Software SA and its affiliates. All rights reserved.
#
set -Eeo pipefail
trap 'finish' SIGTERM SIGHUP SIGINT EXIT

# Check if server is up
healthz()
{
    cmd="agtcmd status"
    out=$($cmd)
    rc=$?
    if [ "$rc" -ne "0" ]; then
        echo "ERR: Event Router not running, rc=$rc, output=$out"
        return -1
    fi
    return 0
}

# Parameters
# $1 -> index
# $2 -> end of env var name
# $3 -> key to be used in file
# $4 -> filename to write on
write_line_in_file()
{
    declare -n env_name="TARGET${1}_${2}"
    if [[ -n "${env_name}" ]]; then
        echo "$3=${env_name}" >> $4
    fi
}

customize_runtime()
{
    echo "INF: Customizing the installation..."

    cp conf/trkagent.ini.template conf/trkagent.ini.template.bck

    # Reset Files
    echo "" > conf/trkagent.def
    echo "" > conf/trkagent.ini
    echo "" > conf/sslconf.ini

    ## General information
    # >> trkagent.def (general)
    echo "[LOG]" >> conf/trkagent.def
    echo "trace=0" >> conf/trkagent.def >> conf/trkagent.def
    echo "log_file=\"$HOME/data/log/log.dat\"" >> conf/trkagent.def
    echo "arc_context_file=\"$HOME/data/log/context_file.txt\"" >> conf/trkagent.def
    echo "nb_arc_file=1" >> conf/trkagent.def
    echo "auto_arc_bytes=10240" >> conf/trkagent.def
    echo "" >> conf/trkagent.def

    # >> trkagent.ini (general)
    echo "[AGENT]" >> conf/trkagent.ini
    echo "name=$ER_NAME" >> conf/trkagent.ini
    echo "target_parameters_file=\"$ER_INSTALLDIR/conf/target.xml\"" >> conf/trkagent.ini
    echo "security_profile_file=\"$ER_INSTALLDIR/conf/sslconf.ini\"" >> conf/trkagent.ini
    if [[ -n "$ER_LOG_LEVEL" ]]; then
        echo "log=$ER_LOG_LEVEL" >> conf/trkagent.ini
    fi
    if [[ -n "$ER_MESSAGE_SIZE" ]]; then
        echo "message_size=$ER_MESSAGE_SIZE" >> conf/trkagent.ini
    fi
    if [[ -n "$ER_RELAY" ]]; then
        if [[ "$ER_RELAY" = "YES" ]]; then
            echo "relay=1" >> conf/trkagent.ini
        else
            echo "relay=0" >> conf/trkagent.ini
        fi
    fi
    echo "" >> conf/trkagent.ini
    echo "[TCPSOURCE]" >> conf/trkagent.ini
    echo "profile=ERSERVER" >> conf/trkagent.ini
    echo "local_address=0.0.0.0" >> conf/trkagent.ini
    if [[ -n "$ER_INCOMING_MAX" ]]; then
        echo "incoming_max=$ER_INCOMING_MAX" >> conf/trkagent.ini
    fi
    if [[ "$ER_USE_SSL" = "NO" ]]; then
        echo "sap=$ER_PORT" >> conf/trkagent.ini
    else
        echo "sapssl=$ER_PORT" >> conf/trkagent.ini

        # >> sslconf.ini (general)
        echo "[ERSERVER]" >> conf/sslconf.ini
        if [[ -n "$ER_CERTIFICATE_FILE" ]]; then
            echo "SSL_USER_CERTIFICATE_FILE = $ER_CERTIFICATE_FILE" >> conf/sslconf.ini
            echo "SSL_USER_CERTIFICATE_FORMAT = PKCS12" >> conf/sslconf.ini
        fi
        if [[ -n "$ER_CERT_PASSWORD_FILE" ]]; then
            echo "SSL_USER_CERTIFICATE_PASSWORD_FILE = $ER_CERT_PASSWORD_FILE" >> conf/sslconf.ini
        fi
        if [[ -n "$ER_SSL_VERIFY_POLICY" ]]; then
            echo "SSL_VERIFY_POLICY = $ER_SSL_VERIFY_POLICY" >> conf/sslconf.ini
        fi
        if [[ -n "$ER_SSL_CIPHER_SUITE" ]]; then
            echo "SSL_CIPHER_SUITE = $ER_SSL_CIPHER_SUITE" >> conf/sslconf.ini
        else
            echo "SSL_CIPHER_SUITE = 156,60,47" >> conf/sslconf.ini
        fi
        if [[ -n "$ER_SSL_VERSION_MIN" ]]; then
            echo "SSL_VERSION_MIN = $ER_SSL_VERSION_MIN" >> conf/sslconf.ini
        else
            echo "SSL_VERSION_MIN = ssl_3.0" >> conf/sslconf.ini
        fi
    fi
    echo "" >> conf/trkagent.ini
    echo "" >> conf/sslconf.ini

    ## Targets' information
    i=1
    declare -n name="TARGET${i}_NAME"
    while [[ -n "${name}" ]]; do
        # >> trkagent.ini (for target)
        echo "[${name}]" >> conf/trkagent.ini
        echo "active=1" >> conf/trkagent.ini
        write_line_in_file "${i}" "LOG_LEVEL" "log" "conf/trkagent.ini"
        write_line_in_file "${i}" "MAX_MESSAGES" "max_messages" "conf/trkagent.ini"
        write_line_in_file "${i}" "TIMEOUT" "timeout" "conf/trkagent.ini"
        write_line_in_file "${i}" "SHORT_WAIT" "short_wait" "conf/trkagent.ini"
        write_line_in_file "${i}" "LONG_WAIT" "long_wait" "conf/trkagent.ini"
        write_line_in_file "${i}" "JUMP_WAIT" "jump_wait" "conf/trkagent.ini"
        write_line_in_file "${i}" "KEEP_CONNECTION" "keep_connection" "conf/trkagent.ini"
        write_line_in_file "${i}" "HEARTBEAT" "heartbeat" "conf/trkagent.ini"
        write_line_in_file "${i}" "PORT" "port" "conf/trkagent.ini"
        write_line_in_file "${i}" "ADDRESS" "address" "conf/trkagent.ini"
        write_line_in_file "${i}" "USE_SSL_OUT" "ssl" "conf/trkagent.ini"
        echo "profile=${name}" >> conf/trkagent.ini
        echo "" >> conf/trkagent.ini
        # >> sslconf.ini (for target)
        declare -n using_ssl="TARGET${i}_USE_SSL_OUT"
        if [[ "${using_ssl}" = "YES" ]]; then
            echo "[${name}]" >> conf/sslconf.ini
            write_line_in_file "${i}" "CA_CERT" "SSL_CA_CERTIFICATE_FILE" "conf/sslconf.ini"
            echo "SSL_CA_CERTIFICATE_FORMAT = PEM" >> conf/sslconf.ini
            write_line_in_file "${i}" "SSL_CIPHER_SUITE" "SSL_CIPHER_SUITE" "conf/sslconf.ini"
            write_line_in_file "${i}" "SSL_VERSION_MIN" "SSL_VERSION_MIN" "conf/sslconf.ini"
            echo "" >> conf/sslconf.ini
        fi

        let i=i+1
        declare -n name="TARGET${i}_NAME"
    done

    ## Default information
    echo "[defaulttarget]" >> conf/trkagent.ini
    echo "directory=\"$HOME/data\"" >> conf/trkagent.ini
    echo "max_messages=$ER_MAX_MESSAGES" >> conf/trkagent.ini

    write_line_in_file "1" "LOG_LEVEL" "log" "conf/trkagent.ini"
    write_line_in_file "1" "MAX_MESSAGES" "max_messages" "conf/trkagent.ini"
    write_line_in_file "1" "TIMEOUT" "timeout" "conf/trkagent.ini"
    write_line_in_file "1" "SHORT_WAIT" "short_wait" "conf/trkagent.ini"
    write_line_in_file "1" "LONG_WAIT" "long_wait" "conf/trkagent.ini"
    write_line_in_file "1" "JUMP_WAIT" "jump_wait" "conf/trkagent.ini"
    write_line_in_file "1" "KEEP_CONNECTION" "keep_connection" "conf/trkagent.ini"
    write_line_in_file "1" "HEARTBEAT" "heartbeat" "conf/trkagent.ini"
    write_line_in_file "1" "PORT" "port" "conf/trkagent.ini"
    write_line_in_file "1" "ADDRESS" "address" "conf/trkagent.ini"
    write_line_in_file "1" "USE_SSL_OUT" "ssl" "conf/trkagent.ini"
    echo "profile=$TARGET1_NAME" >> conf/trkagent.ini

    if [[ -n "$USER_TARGET_XML" ]]; then
        cp $USER_TARGET_XML conf/target.xml
    else
        cp install/target.xml conf/target.xml
    fi

    bin/agtinst -setup -dir $ER_INSTALLDIR -conf $ER_INSTALLDIR/conf/conffile
    cat conf/setup.log

    rm conf/trkagent.ini.template

    echo "INF: installation customized."
}

# Propagating signals
stop()
{
    agtcmd stop
}

kill ()
{
    kill -9 -1
}

finish()
{
    stop
    sleep 2
    kill
}

cd $ER_INSTALLDIR

if [[ "$ER_RECONFIG" = "YES" ]]; then
    if [ -f conf/trkagent.ini.template.bck ]; then
        cp conf/trkagent.ini.template.bck conf/trkagent.ini.template
    fi
fi

if [ -f conf/trkagent.ini.template ]; then
    customize_runtime
else
    echo "INF: installation previously customized."
fi

. conf/profile

# Manage log and output files
ER_LOGDIR=$HOME/data/log
if [ -d $ER_LOGDIR ]; then
    rm -rf $ER_LOGDIR
fi
mkdir $ER_LOGDIR

touch $ER_LOGDIR/AGENT.out
touch $ER_LOGDIR/AGSNTL.out
ln -sf $ER_LOGDIR/AGENT.out tmp/AGENT.out
ln -sf $ER_LOGDIR/AGSNTL.out tmp/AGSNTL.out

if agtcmd start ; then
    echo "INF: start success"
else
    echo "ERR: start returns:"$?
    cat $ER_LOGDIR/log.dat $ER_LOGDIR/AGENT.out $ER_LOGDIR/AGSNTL.out
    exit 1
fi

agtcmd display

# Wait for Event Router to start (and create files)
sleep 10
tail -v $ER_LOGDIR/log.dat -F $ER_LOGDIR/AGENT.out -F $ER_LOGDIR/AGSNTL.out

#### Test that ER is running
maxmumsize=10000
counter=0

healthz
while [ $? -eq 0 ]; do
    sleep ${ER_HEALTHCHECK_INTERVAL:-10}
    healthz

    # Do count every 10 interactions
    if [ $counter %10 ]; then
        agtcmd count
    fi

    # Truncate files when size >= 10M
    actualsize=$(wc -c <"$ER_LOGDIR/AGENT.out")
    if [ $actualsize -ge $minimumsize ]; then
        truncate -s 0 $ER_LOGDIR/AGENT.out
    fi
    actualsize=$(wc -c <"$ER_LOGDIR/AGSNTL.out")
    if [ $actualsize -ge $minimumsize ]; then
        truncate -s 0 $ER_LOGDIR/AGSNTL.out
    fi
    
    ((counter++))
done
