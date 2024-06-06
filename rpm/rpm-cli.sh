#!/bin/bash
#
# shellcheck disable=SC2034 #ignore unused variables


set -eEuo pipefail
shopt -s extdebug

APPNAME=__APPNAME__
APPVERSION="0.0.0"
APPCONF=".env"

trap 'errormsg $LINENO -- ${BASH_LINENO[*]}' ERR

errormsg() {
    error "$0: unexpected error status:$? -- lines:" "$@"
    exit 1
}


# Reset
ColorOff='\033[0m'       # Text Reset

# Regular Colors
Black='\033[0;30m'        # Black
Red='\033[0;31m'          # Red
Green='\033[0;32m'        # Green
Yellow='\033[0;33m'       # Yellow
Blue='\033[0;34m'         # Blue
Purple='\033[0;35m'       # Purple
Cyan='\033[0;36m'         # Cyan
White='\033[0;37m'        # White

# Bold
BBlack='\033[1;30m'       # Black
BRed='\033[1;31m'         # Red
BGreen='\033[1;32m'       # Green
BYellow='\033[1;33m'      # Yellow
BBlue='\033[1;34m'        # Blue
BPurple='\033[1;35m'      # Purple
BCyan='\033[1;36m'        # Cyan
BWhite='\033[1;37m'       # White

# Underline
UBlack='\033[4;30m'       # Black
URed='\033[4;31m'         # Red
UGreen='\033[4;32m'       # Green
UYellow='\033[4;33m'      # Yellow
UBlue='\033[4;34m'        # Blue
UPurple='\033[4;35m'      # Purple
UCyan='\033[4;36m'        # Cyan
UWhite='\033[4;37m'       # White

# Background
On_Black='\033[40m'       # Black
On_Red='\033[41m'         # Red
On_Green='\033[42m'       # Green
On_Yellow='\033[43m'      # Yellow
On_Blue='\033[44m'        # Blue
On_Purple='\033[45m'      # Purple
On_Cyan='\033[46m'        # Cyan
On_White='\033[47m'       # White

# High Intensity
IBlack='\033[0;90m'       # Black
IRed='\033[0;91m'         # Red
IGreen='\033[0;92m'       # Green
IYellow='\033[0;93m'      # Yellow
IBlue='\033[0;94m'        # Blue
IPurple='\033[0;95m'      # Purple
ICyan='\033[0;96m'        # Cyan
IWhite='\033[0;97m'       # White

# Bold High Intensity
BIBlack='\033[1;90m'      # Black
BIRed='\033[1;91m'        # Red
BIGreen='\033[1;92m'      # Green
BIYellow='\033[1;93m'     # Yellow
BIBlue='\033[1;94m'       # Blue
BIPurple='\033[1;95m'     # Purple
BICyan='\033[1;96m'       # Cyan
BIWhite='\033[1;97m'      # White

# High Intensity backgrounds
On_IBlack='\033[0;100m'   # Black
On_IRed='\033[0;101m'     # Red
On_IGreen='\033[0;102m'   # Green
On_IYellow='\033[0;103m'  # Yellow
On_IBlue='\033[0;104m'    # Blue
On_IPurple='\033[0;105m'  # Purple
On_ICyan='\033[0;106m'    # Cyan
On_IWhite='\033[0;107m'   # White

prefix="[$(basename "$0")]"


i=1
sp="/-\|"
sc=0
startspin() {
    sc=0
    printf ' '
}

spin() {
   printf '\r%.1s %s      ' "$sp" "$@" >&2
   sp=${sp#?}${sp%???}
}

endspin() {
   printf "\r%s\n" "$@"
}

dt() {
    local d1
    d1="$(date -Ins --utc)"
    local d2="${d1//+00:00/Z}"
    local d3="${d2//,/.}"
    echo "$d3"
}

error() {
  echo -e "$Red$(dt) $prefix ERR $* $ColorOff" >&2
}

warn() {
  echo -e "$Purple$(dt) $prefix WRN $* $ColorOff" >&2
}

info() {
  echo -e "$Cyan$(dt) $prefix INF $* $ColorOff" >&2
}

debug() {
  echo -e "$Purple$(dt) $prefix DBG $* $ColorOff" >&2
}

nop() {
    echo "" >/dev/null
}

trace() {
  echo -e "$Yellow$(dt) $prefix DBG $* $ColorOff" >&2
  "$@"
}


waitpid () {
    local timeout=10
    local pid=$"$1"
    local start
    local end
    start=$(date +%s)
    end=start
    startspin
    while ps -p "$pid" >/dev/null && (( (end - start) < timeout )); do
        sleep 0.3
        end=$(date +%s)
        spin $(( start + timeout - end ))
    done
    if ps -p "$pid" >/dev/null; then
        endspin "failed"
    else 
        endspin "done"
    fi
}

usage() {
    echo usage
    grep -E "##.*##" "$0" | grep -v "grep" | sed 's/)[^#]*//g' | while IFS="#" read -r cmd a as b msg; do 
        #echo "  $cmd [$as] ($msg)"; 
        printf "%-30.30s %s\n" "$cmd $as" "$msg"
    done
}

verbose=trace
verb=debug

verbose=
verb=nop
NOP=

if [ "$EUID" = 0 ] ; then 
    error "Please run as non root"
    exit 1
fi

INSTALLDIR=$(realpath "$(dirname "$(realpath "$0")")/../..")
if [ "$INSTALLDIR" = "/" ]; then
  INSTALLDIR=""
fi
RUNTIME=""

runtime_env_name="${APPNAME//-/_}_RUNTIME"
runtime_env_name="${runtime_env_name^^}"
$verb "RUNTIME RUNTIME_ENVNAME $runtime_env_name ${!runtime_env_name:-}"
runtime_env_value="${!runtime_env_name:-}"
if [ -n "$runtime_env_value" ]; then 
    RUNTIME="$runtime_env_value"
    $verb "RUNTIME ENV $RUNTIME"
fi

profile="$HOME/.$APPNAME/profile"
if [ -f "$profile" ]; then
    $verb "LOAD PROFILE $profile $(cat "$profile")"
    while read -r line; do
        [ -z "$line" ] && continue
        export "$line";
    done <"$profile"
fi

if [ -z "$RUNTIME" ]; then
    runtime_env_value="${!runtime_env_name:-}"
    if [ -n "$runtime_env_value" ]; then 
        RUNTIME="$runtime_env_value"
        $verb "RUNTIME PROFILE $RUNTIME"
    fi
fi

if [ -z "$RUNTIME" ]; then
    RUNTIME="$(realpath "$(dirname "$0")/../..")"
fi

config_subst() {
    echo "Configuration:"
    echo "-----"
    echo "${runtime_env_name}    runtime folder"
    echo "ER_CONFIG_FILE          path to config file"
    echo "ER_LOG_LEVEL            log level. Supported values: trace, debug, info, warn, error, fatal"
    echo "ER_LOG_USE_LOCALTIME    log uses local time"
    echo "ER_PORT                 http port"
    echo "ER_HOST                 http host"
    echo "ER_LOG_FILE             log file name when writing to a file, leave empty to use standard output"
    echo "ER_LOG_FILE_MAX_SIZE    log file max size (MB) - when writing to a file"
    echo "ER_LOG_FILE_MAX_BACKUP  log file backups - when writing to a file"
    echo "ER_LOG_FILE_MAX_AGE     log file max age (days) - when writing to a file"
    echo "ER_CPU_PROFILE_FILE     write cpu profile to file"
    echo "ER_MEM_PROFILE_FILE     write memory profile to this file"
}

OPTIONS=""
while [ ! $# -eq 0 ] ; do
  case "$1" in
    --version) ## ## Show version
      shift
      echo "$APPVERSION"
      exit 0
    ;;
    -v|--verbose) ## ## Show executed commands
      shift;
      NOP=trace
      verbose=trace
      verb=debug
      OPTIONS+=" --verbose"
    ;;
    --runtime) ## <runtime-dir> ## Use customer runtime folder
        shift;
        RUNTIME="$1"
        shift;
        if [ ! -d "$RUNTIME" ]; then
            error "--runtime $RUNTIME doesn't exists"
            exit 1
        fi
        $verb "RUNTIME --runtime $RUNTIME"
    ;;
    --help) ## ## Get usage
      shift;
      usage
      exit 1
    ;;
    --)
      shift;
      break;
    ;;
    --*)
        error "Invalid flag : $1"
        exit 1
    ;;  
    *)
      break;
  esac
done


#RUNTIME="$(realpath "$RUNTIME")"
#$verb "RUNTIME=$RUNTIME"

# Ensure RUNTIME is properly set to non "/" and does have required files/folders 
case "${1:-}" in
    init)
    ;;
    help)
    ;;
    runtime)
    ;;
    "")
    ;;
    *)
        if [ "$RUNTIME" = "/" ]; then 
            error "RUNTIME cannot be set to '/', you should use $0 init <runtime-folder>, and <runtime-folder>/usr/bin/$(basename "$0") "
            exit 1
        fi

        #if [ ! -d "$RUNTIME/var/lib/volumes" ] ||
        #[ ! -d "$RUNTIME/var/lib/containers" ] ||
        #[ ! -d "$RUNTIME/var/run/states" ]; then
        #    error "RUNTIME badly configured '$RUNTIME'"
        #    exit 1
        #fi
esac

check_pid() {
  PID=$(cat "$RUNTIME/var/run/$APPNAME.pid" 2>/dev/null || echo "")
  if [ -n "$PID" ]; then 
    if ps -p "$PID" > /dev/null; then
      #echo "running ($PID)"
      return 0
    else
      #echo "not running ($PID)"
      return 1
    fi
  else
    #echo "not running"
    return 1
  fi
}

case "${1:-}" in
    start) ## ## start __APPNAME__
      if ! check_pid; then
        cd "$RUNTIME"
        "$INSTALLDIR/usr/bin/${APPNAME}d" --config="$RUNTIME/etc/$APPNAME.conf" > "$RUNTIME/var/log/$APPNAME.log" 2>&1 </dev/null &
        echo "$!" > "$RUNTIME/var/run/$APPNAME.pid"
      else
        echo "$APPNAME is already started"
        exit 1
      fi
    ;;

    stop) ## ## stop __APPNAME__
      if check_pid; then
        PID=$(cat "$RUNTIME/var/run/$APPNAME.pid")
        (sleep 1 && kill "$PID") & 
        waitpid "$PID"
      else
        echo "$APPNAME already stopped"
        exit 1
      fi
    ;;

    restart) ## ## restart __APPNAME__
       cd "$RUNTIME"
      if check_pid; then
        PID=$(cat "$RUNTIME/var/run/$APPNAME.pid")
        (sleep 1 && kill "$PID") & 
        waitpid "$PID"
      else
        echo "$APPNAME is stopped"
        exit 1
      fi
      "$INSTALLDIR/usr/bin/${APPNAME}d" --config="$RUNTIME/etc/$APPNAME.conf" --log-file "$RUNTIME/var/log/$APPNAME.log" >/dev/null 2>&1 </dev/null &
      echo "$!" > "$RUNTIME/var/run/$APPNAME.pid"
    ;;

    logs) ## ## show __APPNAME__ logs
      cat "$RUNTIME/var/log/$APPNAME.log"
    ;;
    
    status) ## ## show the status of __APPNAME__
      PID=$(cat "$RUNTIME/var/run/$APPNAME.pid" 2>/dev/null || echo "")
      echo "$PID"
      if [ -n "$PID" ]; then 
        if ps -p "$PID" > /dev/null; then
          echo "running ($PID)"
          exit 0
        else
          echo "not running ($PID)"
          exit 1
        fi
      else
        echo "not running" 
        exit 1
      fi
    ;;

    live) ## ## show liviness of __APPNAME__
      # FIXME: check curl
      curl -s -v http://localhost:8080/live
    ;;

    ready) ## ## show readiness of __APPNAME__
      # FIXME: check curl
      curl -s -v http://localhost:8080/ready
    ;;

    metrics) ## ## show metrics __APPNAME__
      port=${ER_PORT:-"8080"}
      curl -s http://localhost:$port/metrics
    ;;

    config) ## ## Display the service configuration parameters
      shift
      config_subst
    ;;

    gen-systemd-unit) ## ## Generate a template for systemd
      echo "# /etc/systemd/system/$APPNAME.service"
      echo 
cat <<EOF
[Unit]
Description=Axway Event Router
After=network.target

[Service]
ExecStart=$INSTALLDIR/usr/bin/${APPNAME}d --config=$RUNTIME/etc/${APPNAME}.conf
WorkingDirectory=$RUNTIME
User=$USER
Type=simple

[Install]
WantedBy=default.target
EOF
    ;; 

    init) ## <runtime-dir> ## Initialize a local environment
        shift

        #[ -f "$LIBDIR/preinit.sh" ] && "$LIBDIR/preinit.sh"

        runtime=$(realpath "$1")
        if [ -e "$runtime" ]; then
            error "$runtime already exists"
            exit 1
        fi
        mkdir -p "$runtime"
        mkdir -p "$runtime/etc"
        mkdir -p "$runtime/usr/bin"
        mkdir -p "$runtime/var"
        mkdir -p "$runtime/var/lib"
        mkdir -p "$runtime/var/run"
        mkdir -p "$runtime/var/log"

        #cp -rf "$LIBDIR/etc"/* "$runtime/etc"
        cp -rf "$INSTALLDIR/usr/lib/$APPNAME"/* "$runtime/etc"

        ln -s "$0" "$runtime/usr/bin/$(basename "$0")"

        # create profile
        mkdir -p "$HOME/.$APPNAME"
        oldprofile="$HOME/.$APPNAME/profile.$(date +%F_%H:%M)"
        profile="$HOME/.$APPNAME/profile"

        #save oldprofile
        profilecontent="$(echo "${runtime_env_name}=$(realpath "$runtime")")"
        if [ -f "$profile" ] && [ ! "$(<"$profile")" = "$profilecontent" ]; then
             mv "$profile" "$oldprofile"
        fi

        echo "$profilecontent" > "$profile"
    ;;

    runtime) ## ## Display resolved runtime
        echo "$RUNTIME"
    ;;

    support) ## ## generate a support report
        A="$(pwd)/$APPNAME-support"
        if [ -d "$A" ]; then
          echo "folder $A already exists"
          exit 1
        fi

        mkdir -p "$A"
        ${APPNAME}d help >"$A/help.txt"  || true

        mkdir -p "$A/log"
        cp "$RUNTIME/var/log/$APPNAME.log"* "$A/log"  || true
        
        mkdir -p "$A/etc"
        cp "$RUNTIME/etc/$APPNAME.conf" "$A/etc"  || true

        ps -ef >"$A/ps.txt"  || true 
        find "$RUNTIME" -type f -exec ls -la '{}' \; > "$A/files.txt"
        uname -a > "$A/uname.txt"
        cat /etc/*-release /etc/*_version > "$A/lsb-release.txt" || true
        env > "$A/env.txt"  || true

        find "$A" > "$A/gen.txt"

        tar cvfz "$A.tar.gz" "$A"
        echo "support report generated $A "
    ;;

    help) ## ## Display usage
      usage
    ;;

    *)
        usage
        exit 1
    ;;
esac
