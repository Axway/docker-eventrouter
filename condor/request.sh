#!/bin/bash
#

start=$(($(date +%s) - 86000))000
end=$(($(date +%s) + 86000))000
echo $start $end

TENANTID=jda
# SENDERID
# RECEIVERID
# TRANSMITTEDBYTES
# FLOWNAME

#  "\$match": {"STATE": "SENT"}

echo
echo "Global File Count (Sent)"
curl --show-error -k -s -E certificate.crt --key key-rsa.key -H "Content-Type: application/json" -X POST --data "@-" "https://branch2.trcblt.com:9111/api/v1/compute/?trcbltPartitionId=$TENANTID" <<EOF
{
    "version": "0.2", "invoke": { "method": "count", "field": "@event_time"},
    "filters": {
        "\$range": {"@event_time": {"gte": $start, "lte": $end}},
        "\$match": {"state": "SENT"}
    },
    "group_by": [
        {
            "type": "date",
            "field": "@event_time",
            "params": {
                "default": "2017-01-01T00:00:00.000",
                "interval": "10s"
            }
        }
    ]
}
EOF

echo
echo "Global File Count (Received)"
curl --show-error -k -s -E certificate.crt --key key-rsa.key -H "Content-Type: application/json" -X POST --data "@-" "https://branch2.trcblt.com:9111/api/v1/compute/?trcbltPartitionId=$TENANTID" <<EOF
{
    "version": "0.2", "invoke": { "method": "count", "field": "@event_time"},
    "filters": {
        "\$range": {"@event_time": {"gte": $start, "lte": $end}},
        "\$match": {"state": "RECEIVED"}
    },
    "group_by": [
        {
            "type": "date",
            "field": "@event_time",
            "params": {
                "default": "2017-01-01T00:00:00.000",
                "interval": "10s"
            }
        }
    ]
}
EOF

echo
echo "Global File Count (Errors)"
curl --show-error -k -s -E certificate.crt --key key-rsa.key -H "Content-Type: application/json" -X POST --data "@-" "https://branch2.trcblt.com:9111/api/v1/compute/?trcbltPartitionId=$TENANTID" <<EOF
{
    "version": "0.2", "invoke": { "method": "count", "field": "@event_time"},
    "filters": {
        "\$range": {"@event_time": {"gte": $start, "lte": $end}},
        "\$match": {"state": "CANCELLED"}
    },
    "group_by": [
        {
            "type": "date",
            "field": "@event_time",
            "params": {
                "default": "2017-01-01T00:00:00.000",
                "interval": "10s"
            }
        }
    ]
}
EOF

echo
echo "Global File Volume ( Network Data, SENT)"
curl --show-error -k -s -E certificate.crt --key key-rsa.key -H "Content-Type: application/json" -X POST --data "@-" "https://branch2.trcblt.com:9111/api/v1/compute/?trcbltPartitionId=$TENANTID" <<EOF
{
    "version": "0.2", "invoke": { "method": "sum", "field": "transmittedbytes"},
    "filters": {
        "\$range": {"@event_time": {"gte": $start, "lte": $end}},
        "\$match": {"state": "SENT"}
    },
    "group_by": [
        {
            "type": "date",
            "field": "@event_time",
            "params": {
                "default": "2017-01-01T00:00:00.000",
                "interval": "10s"
            }
        }
    ]
}
EOF

echo
echo "Global File Volume (Filesize, SENT)"
curl --show-error -k -s -E certificate.crt --key key-rsa.key -H "Content-Type: application/json" -X POST --data "@-" "https://branch2.trcblt.com:9111/api/v1/compute/?trcbltPartitionId=$TENANTID" <<EOF
{
    "version": "0.2", "invoke": { "method": "sum", "field": "filesize"},
    "filters": {
        "\$range": {"@event_time": {"gte": $start, "lte": $end}},
        "\$match": {"state": "SENT"}
    },
    "group_by": [
        {
            "type": "date",
            "field": "@event_time",
            "params": {
                "default": "2017-01-01T00:00:00.000",
                "interval": "10s"
            }
        }
    ]
}
EOF


echo 
echo "Getting sum TRANSMITTEDBYTES per RECEIVERID/SENDERID"
curl --show-error -k -s -E certificate.crt --key key-rsa.key -H "Content-Type: application/json" -X POST --data "@-" "https://branch2.trcblt.com:9111/api/v1/compute/?trcbltPartitionId=$TENANTID" <<EOF
{
    "version": "0.2", "invoke": { "method": "sum", "field": "transmittedbytes"}, 
    "filters": {
        "\$range": {"@event_time": {"gte": $start, "lte": $end}},
        "\$match": {"state": "SENT"}
    },
    "group_by": [
        "RECEIVERID", "senderid",
        {
            "type": "date",
            "field": "@event_time",
            "params": {
                "default": "2017-01-01T00:00:00.000",
                "interval": "1s"
            }
        }
    ]
}
EOF

echo
echo "Getting COUNT per STATE"
curl --show-error -k -s -E certificate.crt --key key-rsa.key -H "Content-Type: application/json" -X POST --data "@-" "https://branch2.trcblt.com:9111/api/v1/compute/?trcbltPartitionId=$TENANTID" <<EOF
{
    "version": "0.2", "invoke": { "method": "count", "field": "@event_time"}, 
    "filters": {
        "\$range": {"@event_time": {"gte": $start, "lte": $end}}
    },
    "group_by": [
        "state"
    ]
}
EOF

echo
echo "Getting Errors"
curl --show-error -k -s -E certificate.crt --key key-rsa.key -H "Content-Type: application/json" -X POST --data "@-" "https://branch2.trcblt.com:9111/api/v1/search/?trcbltPartitionId=$TENANTID" <<EOF
{
    "version": "0.2",
    "invoke": {
        "method": "find",
        "params":  {
            "limit": 10
        }
    },
    "filters": {
        "\$range": {
            "@event_time": {
                "gte": $start,
                "lte": $end
            }
        },
        "\$match" : { "state": "CANCELED"}
    }
}
EOF

echo
echo "Getting Sent messages (for debugging....)"
curl --show-error -k -s -E certificate.crt --key key-rsa.key -H "Content-Type: application/json" -X POST --data "@-" "https://branch2.trcblt.com:9111/api/v1/search/?trcbltPartitionId=$TENANTID" <<EOF
{
    "version": "0.2",
    "invoke": {
        "method": "find",
        "params":  {
            "limit": 10
        }
    },
    "filters": {
        "\$range": {
            "@event_time": {
                "gte": $start,
                "lte": $end
            }
        },
        "\$match" : { "STATE": "SENT"}
    }
}
EOF

echo
echo "Getting Latest messages (for debugging....)"
curl --show-error -k -s -E certificate.crt --key key-rsa.key -H "Content-Type: application/json" -X POST --data "@-" "https://branch2.trcblt.com:9111/api/v1/search/?trcbltPartitionId=$TENANTID" <<EOF
{
    "version": "0.2",
    "invoke": {
        "method": "find",
        "params":  {
            "limit": 10
        }
    },
    "filters": {
        "\$range": {
            "@event_time": {
                "gte": $start,
                "lte": $end
            }
        }
    }
}
EOF