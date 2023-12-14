# event-router


https://docs.axway.com/bundle/amplify-central/page/docs/integrate_with_central/query-the-traceability-apis-with-lexus/index.html


SWPOSSIBLEDUPLICATEIND
## Use Cases

Classical QLT Event Router :
    - store and forward queue (push-push) QLTServer -> store -> QLTClient
    - store and forward queue (push-pull / pull-push) (DMZ)
    - filtering
    - dispatching
    - retries
    - usage local file storage
    - Limits : 
      - storage : (fs required) (cloud native?)
      - speed: no pipelining
      - resilience : may lose one message / may duplicate 
      - security :
      - messages : xml : no json
  
Product -> ER -WAN-> ER -> Sentinel

Additional features/usage case
    - [x] observability through prometheus
    - Buffering (store/forward)
      - [x] DB(postgres...) usage as buffer
      - [x] Kafka usage as buffer
    - Transform
      - [x] json
    - Event End storage
      - Elasticsearch (json)
      - [x] Mongodb (json)
      - DB (json)
      - DB (Sentinel TO)
      - Kafka (json)
      - Localfile (json)
    - Simple Event aggregation
      - usage tracking (CFT/ST) (dedicated limited buffer : )
      - combined cycleid events (AKA current like)
      - combined cycling/cycle events
    - Protocol
      - line compression
      - pipelining with deduplication? 

- deduplication

## Features
- listen to qlt protocol QLT_PORT / QLT_HOST
- filter QLT empty fields : "" 0 
- route QLT messages to
    - Elasticsearch 7 (API) : ELASTISEARCH_URL
    - Logstash (Lumberjack): LUMBERJACK_ADDR
    - Sentinel (QLT) : SENTINEL_ADDR untransformed
    - Localfile : FILENAME
- observability through prometheus

# Changelog
- 0.0.4
    - fix attribute name case for TrkIdentifier
    - add version,build information when starting
    - do not start all services (qlt, qlts) by default
- 0.0.3
    - support XML encoding (8859-1 for example)
    - fix "<" in XML attributes instead of &lt;
    - support multiple targets for sentinel_addrs and count
- 0.0.2
    - add TLS support for QLT server
    - add TLS support for lumberjack
    - add initial k8s/helm support

# Limitations
- Unreliable message routing (message can be lost...)
- No support for external QLT port in helm

# Todo
- Refactor TLS
- Add Non transforming QLT relay
- Add cusotmization for tenant (QLT/LumberJack)
- Message transformation : Configurable QLT message transformation
    (numeric, date, uppercase)
- Message Filtering
- Reliability
    - End-to-End Ack support
    - Intermediate Buffer
- Ingest pipelining ?

## Cloud environment (helm)
- curl http://qlt-router.eks.mft.apirs.net
- helm list
- helm install ./helm/event-router --name event-router --namespace jda
- helm del --purge event-router
- kubectl get svc,pods
- kubectl logs -lapp.kubernetes.io/name=event-router


```sh
make && ./event-router --qlt_port=3333 --filename="zouzou" "--pg_url=postgresql://mypguser:mypgsecretpassword@localhost:5432/mypgdb"
```

# 

current:
  - xml / json
  - connectors
    - qlt
    - lumberjack
    - es
    - mongodb
    - kafka
    - postgres
    - file

  platform:
    - linux/windows/macos

  feature:
    - pipelining
    - 

limits:
- retries ?
- dispatching ?
- filtering ?

- websocket
- pull mode
  
- TLSv3: security

- ci/cd
- configuration
- documentation
- QA*

questions ?
- migration from event router
- basic use cases:

- use cases: