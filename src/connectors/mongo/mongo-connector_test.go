package mongoConnector

import (
	"testing"

	"axway.com/qlt-router/src/connectors/memtest"
	"github.com/a8m/envsubst"
)

func TestMongoConnectorGen(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
		return
	}
	t.Parallel()

	url, _ := envsubst.String("mongodb://root:mymongosecret@${MONGO:-localhost}:27017/")
	err := mongoDBInitFromUrl(t.Name(), url)
	if err != nil {
		t.Fatal("Error cleaning up database", err)
		return
	}
	writer := &MongoWriterConf{
		Url:        url,
		Db:         "mymongodb",
		Collection: "buffer",
	}
	reader := &MongoReaderConf{
		Url:               url,
		Db:                "mymongodb",
		Collection:        "buffer",
		ReadersCollection: "bufferReaders",
		ReaderName:        "test",
	}
	memtest.TestConnector(t, writer, reader)
}
