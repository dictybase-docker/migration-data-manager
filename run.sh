#!/bin/sh

app --bp so --bp oborel --bp eco --bp cl --gh

if [ ${ETCD_CLIENT_SERVICE_HOST+defined} ]
then
    curl http://${ETCD_CLIENT_SERVICE_HOST}:${ETCD_CLIENT_SERVICE_PORT}/v2/keys/migration/download -XPUT -d value="complete"
else
    echo host not found
fi
