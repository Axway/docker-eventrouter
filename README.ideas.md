# ER / Metrics

## Principles

- data resilience/safety/integrity
- scalability / cloud readiness
- 

## Data storage

- kafka
- DB (postgres, mariadb...) sqlserver?, !oracle


## functional

- status 
- live count

## model

Producer -> Processor -> Consumer

Producers
    - QLTServer
    - File : json/raw/DB

Consumers
    - QLTClient
    - DBJson
    - DBEvent
    - ES
    - Kafka

Processor
    - Filter
    - Enrich
    - ConvertJSON
    - Split?

StoreAndForward

Question
    - single vs multiple messages
    - ack management : ack id, ack count...
    - ack origin/backtrack
    - intermediate buffer/store + notification
    - optimizations : 
      - (overuse of) channels vs coalesce function (consumer / producer)
      - multiply streams
      - channel sizes

## Configuration


P (wMsgQueue, rAckqueue, conf)
T (rMsgQueue, wAckqueue, rMsgQueues, wAckQueues, conf)
F (rMsgQueue, aAckQueue, rMsgQueues, wAckQueues, conf)
C (rMsgQUeue, aAckQueue, conf)


entities
    name:
    type:
        conf

streams
    S1 :
        P1 -> F1 -> C1
        P2 -> F2 -> F3 -> C1
