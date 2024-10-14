package repository

import (
	"database/sql"
	"time"
)

type LotteryRepository struct {
	DB *sql.DB
}

func (lr *LotteryRepository) GetLotteryByShortNumber(shortNumber string, currentDateTime time.Time) (int64, string, string, error) {
	var id int64
	var code, answer string
	query := `
        SELECT l.id, l.sms_code, l.sms_answer
        FROM lotteries l
        JOIN accounts a ON l.account_id = a.id
        WHERE a.short_number = ? AND l.start_time <= ? AND l.end_time >= ?
    `
	err := lr.DB.QueryRow(query, shortNumber, currentDateTime, currentDateTime).Scan(&id, &code, &answer)
	if err != nil {
		return 0, "", "", err
	}
	return id, code, answer, nil
}

func (lr *LotteryRepository) InsertLotteryMessageAndUpdate(id int64, message string, parsedDate time.Time, clientID int64) error {

	_, err := lr.DB.Exec(
		"INSERT INTO lottery_sms_messages (lottery_id, msg, dt, client_id) VALUES (?, ?, ?, ?)",
		id, message, parsedDate, clientID,
	)
	return err
}
