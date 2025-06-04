package postgres

import "database/sql"

type QLTRow struct {
	id   int64
	text string
}

func pgDBGetLast(conn *sql.DB, tab string) (int64, error) {
	var count sql.NullInt64
	err := conn.QueryRow("SELECT MAX(id) FROM " + tab).Scan(&count)
	if err != nil {
		return 0, err
	}
	if !count.Valid {
		return 0, nil
	}
	return count.Int64, nil
}

func pgDBRead(conn *sql.DB, maxlength int, offset int, tab string) ([]QLTRow, error) {
	qrows, err := conn.Query("SELECT id, name FROM "+tab+"  WHERE id > $1 ORDER BY id LIMIT $2", offset, maxlength)
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
