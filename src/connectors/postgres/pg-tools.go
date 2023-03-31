package postgres

import "database/sql"

type QLTRow struct {
	id   int64
	text string
}

func pgDBGetLast(conn *sql.DB) (int64, error) {
	var count int64
	err := conn.QueryRow("SELECT MAX(id) FROM " + QLTTABLE).Scan(&count)
	return count, err
}

func pgDBRead(conn *sql.DB, maxlength int, offset int) ([]QLTRow, error) {
	qrows, err := conn.Query("SELECT * FROM "+QLTTABLE+"  WHERE id > $2 ORDER BY id LIMIT $1", maxlength, offset)
	if err != nil {
		return nil, err
	}
	defer qrows.Close()
	var trows []QLTRow

	for qrows.Next() {
		var row QLTRow
		if err := qrows.Scan(&row.id, &row.text); err != nil {
			return trows, err
		}
		trows = append(trows, row)
	}
	if err = qrows.Err(); err != nil {
		return trows, err
	}
	return trows, nil
}
