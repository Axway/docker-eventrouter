package postgres

import (
	"testing"

	"axway.com/qlt-router/src/connectors/memtest"
	"github.com/a8m/envsubst"
)

func TestPGConnectorGen(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
		return
	}
	t.Parallel()

	url, _ := envsubst.String("postgresql://mypguser:mypgsecretpassword@${POSTGRESQL:-localhost}:5432/mypgdb")
	err := pgDBInitFromUrl(t.Name(), url)
	if err != nil {
		t.Fatal("Error cleaninup database", err)
		return
	}
	writer := &PGWriterConf{Url: url, Initialize: false, Table: "QLTBuffer"}
	reader := &PGReaderConf{Url: url, ReaderName: "testAuto", Table: "QLTBuffer"}
	memtest.TestConnector(t, writer, reader)
}
