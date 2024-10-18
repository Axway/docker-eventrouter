#!/bin/bash
#

HELM_NAME="eventrouter"

ER_IMAGE_REPOSITORY="docker.repository.axway.com/sentineleventrouter-docker-prod/3.0/eventrouter"
ER_IMAGE_TAG="3.0.20240529"

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
        DEBUG helm upgrade --install "$HELM_NAME" ./event-router --set image.repository=${ER_IMAGE_REPOSITORY},image.tag=${ER_IMAGE_TAG}
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
        DEBUG helm upgrade --install "$HELM_NAME" ./event-router --set image.repository=${ER_IMAGE_REPOSITORY},image.tag=${ER_IMAGE_TAG}
    ;;

    "status")
        DEBUG kubectl get statefulset/"$HELM_NAME"-event-router
    ;;

    "inspect")
        DEBUG kubectl describe statefulset/"$HELM_NAME"-event-router
        DEBUG kubectl describe service/"$HELM_NAME"-event-router
    ;;

    "logs")
        DEBUG kubectl logs statefulset/"$HELM_NAME"-event-router
    ;;

    *)
        if [ ! -z "${1:-}" ]; then
            echo "unsupported command $1"
        fi
        echo "$0 create | delete | replace | status | inspect | logs | wait-ready | wait-delete"
    ;;
esac
