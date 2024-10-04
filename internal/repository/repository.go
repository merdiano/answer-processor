package repository

import "database/sql"

func GetAccountType(db *sql.DB, shortNumber string) (string, error) {
	var accountType string
	err := db.QueryRow("SELECT type FROM accounts WHERE short_number = ?", shortNumber).Scan(&accountType)
	if err != nil {
		return "", err
	}
	return accountType, nil
}

func InsertClientIfNotExists(db *sql.DB, phoneNumber string) (int64, error) {
	var id int64
	err := db.QueryRow("SELECT id FROM clients WHERE phone = ?", phoneNumber).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	if id == 0 {
		result, err := db.Exec("INSERT INTO clients (phone, created_at, updated_at) VALUES (?, NOW(), NOW())", phoneNumber)
		if err != nil {
			return 0, err
		}
		id, err = result.LastInsertId()
		if err != nil {
			return 0, err
		}
		loggers.InfoLogger.Info("Inserted new client", "phone", phoneNumber)
	} else {
		loggers.InfoLogger.Info("Client already exists", "phone", phoneNumber)
	}
	return id, nil
}
