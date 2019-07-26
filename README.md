# qlt-router

## Features
- listen to qlt protocol QLT_PORT / QLT_HOST
- filter QLT empty fields : "" 0 
- route QLT messages to
    - Elasticsearch 7 (API) : ELASTISEARCH_URL
    - Logstash (Lumberjack): LUMBERJACK_ADDR
    - Sentinel (QLT) : SENTINEL_ADDR
    - Localfile : FILENAME
- observability through prometheus

# Limitations
- No TLS support
- Unreliable message routing (message can be lost...)
- No support for external QLT port in helm

# Todo
- TLS Support
- Mutual TLS Support
- Message transformation : Configurable QLT message transformation
- Message Filtering
- Reliability
    - End-to-End Ack support
    - Intermediate Buffer
- Ingest pipelining ?

## Cloud environment (helm)
- curl http://qlt-router.eks.mft.apirs.net
- helm list
- helm install ./helm/qlt-router --name qlt-router --namespace jda
- helm del --purge qlt-router
- kubectl get svc,pods
- kubectl logs -lapp.kubernetes.io/name=qlt-router
