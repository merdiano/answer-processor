package repository

import (
	"database/sql"
	"time"
)

type ShopRepository struct {
	DB *sql.DB
}

func (sr *ShopRepository) GetLotDetailsByShortNumber(shortNumber string, currentDateTime time.Time) (int64, string, error) {
	var lotID int64
	var description string
	query := `
        SELECT l.id, l.description
        FROM lots l
        JOIN accounts a ON l.account_id = a.id
        WHERE a.short_number = ? AND l.starts_at <= ? AND l.ends_at >= ?
    `
	err := sr.DB.QueryRow(query, shortNumber, currentDateTime, currentDateTime).Scan(&lotID, &description)
	if err != nil {
		return 0, "", err
	}
	return lotID, description, nil
}

func (sr *ShopRepository) InsertLotMessageAndUpdate(lotID int64, msg string, dt time.Time, clientID int64) error {
	tx, err := sr.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		"INSERT INTO lot_sms_messages (lot_id, msg, dt, client_id) VALUES (?, ?, ?, ?)",
		lotID, msg, dt, clientID,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
