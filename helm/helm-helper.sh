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
        DEBUG helm upgrade --install "$HELM_NAME" ./event-router --set image.repository=eventrouter/eventrouter,image.tag=2.4.0-SP3
    ;;

    "delete")
        DEBUG helm delete $HELM_NAME
    ;;

    "wait-started")
        DEBUG kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=event-router --timeout=10s
    ;;

    "wait-delete")
        DEBUG kubectl wait --for=delete pod -l app.kubernetes.io/name=event-router --timeout=10s
    ;;

    "replace")
        DEBUG helm upgrade --install "$HELM_NAME" ./event-router --set image.repository=eventrouter/eventrouter,image.tag=2.4.0-SP3
    ;;

    "status")
        DEBUG kubectl get statefulset/event-router
    ;;

    "inspect")
        DEBUG kubectl describe statefulset/event-router
        DEBUG kubectl describe service/event-router
    ;;

    "logs")
        DEBUG kubectl logs statefulset/event-router
    ;;

    *)
        if [ ! -z "${1:-}" ]; then
            echo "unsupported command $1"
        fi
        echo "$0 create | delete | replace | status | inspect | logs | wait-ready"
    ;;
esac
