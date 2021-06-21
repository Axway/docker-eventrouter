#!/bin/bash
#
# DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS HEADER.
#
# Copyright (c) 2021 Axway Software SA and its affiliates. All rights reserved.
#
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
    if [[ "${1}" -eq "0" ]] ; then
        declare -n env_name="DEFAULT_${2}"
        if [[ -z "${env_name}" ]]; then
            declare -n env_name="TARGET1_${2}"
        fi
    else
        declare -n env_name="TARGET${1}_${2}"
    fi

    if [[ -n "${env_name}" ]]; then
        echo "$5" "$3=$6${env_name}$6" >> "$4"
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
    echo "[INFO]" >> conf/trkagent.def
    echo "docker=1" >> conf/trkagent.def
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
        ER_RELAY=`echo $ER_RELAY | tr '[a-z]' '[A-Z]'`
        if [[ "$ER_RELAY" = "YES" ]]; then
            echo "relay=1" >> conf/trkagent.ini
        else
            echo "relay=0" >> conf/trkagent.ini
        fi
    fi
    echo "" >> conf/trkagent.ini
    echo "[TCPSOURCE]" >> conf/trkagent.ini
    echo "local_address=0.0.0.0" >> conf/trkagent.ini
    if [[ -n "$ER_INCOMING_MAX" ]]; then
        echo "incoming_max=$ER_INCOMING_MAX" >> conf/trkagent.ini
    fi

    if [[ -n "$ER_USE_SSL" ]]; then
        ER_USE_SSL=`echo $ER_USE_SSL | tr '[a-z]' '[A-Z]'`
    fi
    if [[ "$ER_USE_SSL" = "YES" ]]; then
        echo "profile=ERSERVER" >> conf/trkagent.ini
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
    else
        echo "sap=$ER_PORT" >> conf/trkagent.ini
    fi
    echo "" >> conf/trkagent.ini
    echo "" >> conf/sslconf.ini

    ## target section
    echo "[sentinel]" >> conf/trkagent.ini
    echo "active=1" >> conf/trkagent.ini
    write_line_in_file "0" "LOG_LEVEL" "log" "conf/trkagent.ini" "" ""
    echo "" >> conf/trkagent.ini

    ## Default information
    echo "[defaulttarget]" >> conf/trkagent.ini
    echo "directory=\"$HOME/data\"" >> conf/trkagent.ini
    write_line_in_file "0" "LOG_LEVEL" "log" "conf/trkagent.ini" "" ""
    write_line_in_file "0" "MAX_MESSAGES" "max_messages" "conf/trkagent.ini" "" ""
    write_line_in_file "0" "TIMEOUT" "timeout" "conf/trkagent.ini" "" ""
    write_line_in_file "0" "SHORT_WAIT" "short_wait" "conf/trkagent.ini" "" ""
    write_line_in_file "0" "LONG_WAIT" "long_wait" "conf/trkagent.ini" "" ""
    write_line_in_file "0" "JUMP_WAIT" "jump_wait" "conf/trkagent.ini" "" ""
    write_line_in_file "0" "KEEP_CONNECTION" "keep_connection" "conf/trkagent.ini" "" ""
    write_line_in_file "0" "HEARTBEAT" "heartbeat" "conf/trkagent.ini" "" ""
    write_line_in_file "0" "PORT" "port" "conf/trkagent.ini" "" ""
    write_line_in_file "0" "ADDRESS" "address" "conf/trkagent.ini" "" ""
    write_line_in_file "0" "BACKUP_PORT" "backup_port" "conf/trkagent.ini" "" ""
    write_line_in_file "0" "BACKUP_ADDRESS" "backup_address" "conf/trkagent.ini" "" ""
    write_line_in_file "0" "USE_SSL_OUT" "ssl" "conf/trkagent.ini" "" ""

    DEFAULT_USE_SSL_OUT=$(echo $DEFAULT_USE_SSL_OUT | tr '[a-z]' '[A-Z]')
    if [[ "$DEFAULT_USE_SSL_OUT" = "YES" ]]; then
        echo "profile=DEFAULT" >> conf/trkagent.ini

        # >> sslconf.ini (general)
        echo "[DEFAULT]" >> conf/sslconf.ini
        if [[ -n "$DEFAULT_CA_CERT" ]]; then
            echo "SSL_CA_CERTIFICATE_FILE = $DEFAULT_CA_CERT" >> conf/sslconf.ini
            echo "SSL_CA_CERTIFICATE_FORMAT = PEM" >> conf/sslconf.ini
        fi
        if [[ -n "$DEFAULT_SSL_CIPHER_SUITE" ]]; then
            echo "SSL_CIPHER_SUITE = $DEFAULT_SSL_CIPHER_SUITE" >> conf/sslconf.ini
        fi
        if [[ -n "$DEFAULT_SSL_VERSION_MIN" ]]; then
            echo "SSL_VERSION_MIN = $DEFAULT_SSL_VERSION_MIN" >> conf/sslconf.ini
        fi
        echo "" >> conf/sslconf.ini
    fi

    if [[ -z "$USER_TARGET_XML" ]]; then
        # Create our target.xml
        echo "" > conf/target.xml
        echo "<TrkEventRouterCfg>" >> conf/target.xml
        echo "   <TrkXml version=\"1.0\" />" >> conf/target.xml
        echo "   <EventRouter name=\"DEFAULT\">" >> conf/target.xml
        echo "   </EventRouter>" >> conf/target.xml


        ## Targets' information
        i=1
        declare -n name="TARGET${i}_NAME"
        while [[ -n "${name}" ]]; do
            declare -n using_ssl="TARGET${i}_USE_SSL_OUT"
            if [[ -n "${using_ssl}" ]]; then
                using_ssl=$(echo "${using_ssl}" | tr '[a-z]' '[A-Z]')
            fi

            # >> target entry in target.xml
            echo -n "   <Target name=\"${name}\"" >> conf/target.xml
            write_line_in_file "${i}" "HEARTBEAT" " heartbeat" "conf/target.xml" "-n" "\""
            echo -n " defaultXntf=\"yes\"" >> conf/target.xml
            echo " defaultXml=\"yes\">" >> conf/target.xml

            echo -n "     <Access" >> conf/target.xml
            write_line_in_file "${i}" "PORT" " port" "conf/target.xml" "-n" "\""
            write_line_in_file "${i}" "ADDRESS" " addr" "conf/target.xml" "-n" "\""
            write_line_in_file "${i}" "BACKUP_PORT" " backup_port" "conf/target.xml" "-n" "\""
            write_line_in_file "${i}" "BACKUP_ADDRESS" " backup_addr" "conf/target.xml" "-n" "\""
            if [[ "${using_ssl}" = "YES" ]]; then
                echo -n " ssl=\"yes\"" >> conf/target.xml
                echo -n " profile=\"${name}\"" >> conf/target.xml
            else
                echo -n " ssl=\"no\"" >> conf/target.xml
            fi
            echo " />" >> conf/target.xml

            echo -n "     <Connection" >> conf/target.xml
            write_line_in_file "${i}" "SHORT_WAIT" " short_wait" "conf/target.xml" "-n" "\""
            write_line_in_file "${i}" "JUMP_WAIT" " jump_wait" "conf/target.xml" "-n" "\""
            write_line_in_file "${i}" "LONG_WAIT" " long_wait" "conf/target.xml" "-n" "\""
            write_line_in_file "${i}" "KEEP_CONNECTION" " keep_connection" "conf/target.xml" "-n" "\""
            write_line_in_file "${i}" "TIMEOUT" " timeout" "conf/target.xml" "-n" "\""
            echo " />" >> conf/target.xml

            echo -n "     <File" >> conf/target.xml
            write_line_in_file "${i}" "MAX_MESSAGES" " max_messages" "conf/target.xml" "-n" "\""
            echo " />" >> conf/target.xml

            echo "   </Target>" >> conf/target.xml

            # >> sslconf.ini (for target)
            if [[ "${using_ssl}" = "YES" ]]; then
                echo "[${name}]" >> conf/sslconf.ini
                write_line_in_file "${i}" "CA_CERT" "SSL_CA_CERTIFICATE_FILE" "conf/sslconf.ini" "" ""
                echo "SSL_CA_CERTIFICATE_FORMAT = PEM" >> conf/sslconf.ini
                write_line_in_file "${i}" "SSL_CIPHER_SUITE" "SSL_CIPHER_SUITE" "conf/sslconf.ini" "" ""
                write_line_in_file "${i}" "SSL_VERSION_MIN" "SSL_VERSION_MIN" "conf/sslconf.ini" "" ""
                echo "" >> conf/sslconf.ini
            fi

            let i=i+1
            declare -n name="TARGET${i}_NAME"
        done

        echo "</TrkEventRouterCfg>" >> conf/target.xml
    else
        # Use target.xml given by client
        cp $USER_TARGET_XML conf/target.xml
    fi

    echo "=============== conf/trkagent.ini"
    cat conf/trkagent.ini
    echo "=============== conf/sslconf.ini"
    cat conf/sslconf.ini
    echo "=============== conf/target.xml"
    cat conf/target.xml
    echo "=============== conf/trkagent.def"
    cat conf/trkagent.def

    bin/agtinst -setup -dir $ER_INSTALLDIR -conf $ER_INSTALLDIR/conf/conffile
    if [ $? -ne 0 ]; then
        cat conf/setup.log
        echo "ERR: Failed to configure Event Router"
        return -1
    fi
    echo "=============== conf/setup.log"
    cat conf/setup.log

    rm conf/trkagent.ini.template

    echo "INF: installation customized."
}

# Propagating signals
stop()
{
    agtcmd stop
    if [ $? -eq 0 ]; then
        exit 0
    fi
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

cd "$ER_INSTALLDIR"

ER_RECONFIG=$(echo $ER_RECONFIG | tr '[a-z]' '[A-Z]')
if [[ "$ER_RECONFIG" = "YES" ]]; then
    if [ -f conf/trkagent.ini.template.bck ]; then
        cp conf/trkagent.ini.template.bck conf/trkagent.ini.template
    fi
fi

if [ -f conf/trkagent.ini.template ]; then
    customize_runtime
    if [ $? -ne 0 ]; then
        echo "ERR: Failed to customize the runtime"
        exit 1
    fi
else
    echo "INF: installation previously customized."
fi

. conf/profile

if agtcmd start ; then
    echo "INF: start success"
else
    echo "ERR: start returns:"$?
    exit 1
fi

# Wait for Event Router to start (and create files)
# wait startup
timeout=30
i=0
echo "Waiting for event router startup $i/$timeout..."
while [ $i -lt $timeout ]; do
  healthz
  status_rc=$?
  if [ "$status_rc" = "0" ]; then
    i=$timeout
  else
    i=$(($i+5))
    sleep 5
    echo "Waiting for event router startup $i/$timeout..."
  fi
done


agtcmd display

#### Test that ER is running
counter=0

healthz
while [ $? -eq 0 ]; do
    sleep ${ER_HEALTHCHECK_INTERVAL:-10}

    # Do count every 10 interactions
    if ! (( $counter%10 )) ; then
        agtcmd count
        if [ $? -ne 0 ]; then
            echo "ERR: Failed to execute agtcmd count"
            exit 1
        fi
    fi

    ((counter++))
    healthz
done
