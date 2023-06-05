package main

import (
	"database/sql"
	"fmt"
	"os"

	"axway.com/qlt-router/src/log"
	_ "github.com/godror/godror"
)

const (
	hostname = "10.128.131.139"
	port     = 1521
)

// const hostname = "52.136.219.167"
// const port = 1521
const service = "PDBBP2I"

// const service = "oracle19c"
const (
	username = "SNTLBP2I"
	password = "SNTLBP2I"
)

type MetalScanner struct {
	valid bool
	value interface{}
}

func (scanner *MetalScanner) getBytes(src interface{}) []byte {
	if a, ok := src.([]uint8); ok {
		return a
	}
	return nil
}

func (scanner *MetalScanner) Scan(src interface{}) error {
	scanner.value = src
	scanner.valid = true
	return nil
	/*
		switch src.(type) {
		case int64:
			if value, ok := src.(int64); ok {
				scanner.value = value
				scanner.valid = true
			}
		case float64:
			if value, ok := src.(float64); ok {
				scanner.value = value
				scanner.valid = true
			}
		case bool:
			if value, ok := src.(bool); ok {
				scanner.value = value
				scanner.valid = true
			}
		case string:
			scanner.value = src
			scanner.valid = true
		case []byte:
			value := scanner.getBytes(src)
			scanner.value = value
			scanner.valid = true
		case time.Time:
			if value, ok := src.(time.Time); ok {
				scanner.value = value
				scanner.valid = true
			}
		case nil:
			scanner.value = nil
			scanner.valid = true
		default:
			//value := scanner.getBytes(src)
			//scanner.value = string(value)
			//scanner.valid = true
			fmt.Println("oups", reflect.TypeOf(src), src)
		}
		return nil*/
}

func oracle() {
	connectString := fmt.Sprintf(`user="%s" password="%s" connectString="%s:%d/%s" timeout="3"`, username, password, hostname, port, service)
	log.Infoln("Oracle connect-string", connectString)

	db, err := sql.Open("godror", connectString)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer db.Close()

	/*ctx, stop := context.WithCancel(context.Background())
	defer stop()

	log.Infoln("Oracle Ping", connectString)
	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
		return
	}*/

	log.Infoln("Oracle Query", connectString)
	rows, err := db.Query("SELECT * FROM HISTORIC_664943629 FETCH FIRST 5 ROWS ONLY")
	if err != nil {
		log.Errorln("Error running query", err)
		return
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	log.Println("Oracle Query Columns", cols)

	log.Infoln("Oracle Query Response")
	for rows.Next() {
		// rows.Scan(&thedate)
		row := make([]interface{}, len(cols))
		for idx := range cols {
			row[idx] = new(MetalScanner)
		}

		err := rows.Scan(row...)
		if err != nil {
			fmt.Println(err)
		}
		for idx, column := range cols {
			scanner := row[idx].(*MetalScanner)

			val := fmt.Sprint(scanner.value)
			if val != "" && val != "<nil>" {
				fmt.Println(column, ":", val)
			}

		}
	}

	os.Exit(1)
}
