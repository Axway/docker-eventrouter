#!/bin/bash
#

HELM_NAME="event-router"

set -uoe pipefail

COL_MSG="\033[92m"
COL_CLEAR="\033[0m"

DEBUG() {
    echo
    echo -e "$COL_MSG> $@$COL_CLEAR"
    $@
}

if [ -f "./.env" ]; then
    . ./.env
fi

case "${1:-}" in
    "create")
        DEBUG kubectl get secrets
        DEBUG helm upgrade --install "$HELM_NAME" ./event-router --set image.repository=eventrouter/eventrouter,image.tag=2.4.0-SP4
    ;;

    "delete")
        DEBUG helm delete $HELM_NAME
    ;;

    "wait-ready")
        DEBUG kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=event-router --timeout=10s
    ;;

    "wait-delete")
        DEBUG kubectl wait --for=delete pod -l app.kubernetes.io/name=event-router --timeout=10s
    ;;

    "replace")
        DEBUG helm upgrade --install "$HELM_NAME" ./event-router --set image.repository=eventrouter/eventrouter,image.tag=2.4.0-SP4
    ;;

    "status")
        DEBUG kubectl get statefulset/"$HELM_NAME"
    ;;

    "inspect")
        DEBUG kubectl describe statefulset/"$HELM_NAME"
        DEBUG kubectl describe service/"$HELM_NAME"
    ;;

    "logs")
        DEBUG kubectl logs statefulset/"$HELM_NAME"
    ;;

    *)
        if [ ! -z "${1:-}" ]; then
            echo "unsupported command $1"
        fi
        echo "$0 create | delete | replace | status | inspect | logs | wait-ready | wait-delete"
    ;;
esac
