package sqlite

import (
	"os"
	"testing"

	"axway.com/qlt-router/src/connectors/memtest"
)

func TestSqliteConnectorGen(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
		return
	}
	t.Parallel()

	path := "./test.db"
	defer os.Remove(path)
	err := sqliteDBInitFromPath(t.Name(), path)
	if err != nil {
		t.Fatal("Error cleaninup database", err)
		return
	}
	writer := &SqliteWriterConf{Path: path, Initialize: false, Table: "QLTBuffer"}
	reader := &SqliteReaderConf{Path: path, ReaderName: "testAuto", Table: "QLTBuffer"}
	memtest.TestConnector(t, writer, reader)
}
