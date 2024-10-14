package repository

import (
	"database/sql"

	// "errors"
	// "strings"
	"time"
)

type QuestionScoringInfo struct {
	ID                         int64  // Question ID
	QuizID                     int64  // Quiz ID
	Answer                     string // Question answer
	Score                      int    // Question score
	HasScored                  bool   // Whether the client has scored or not
	HasMistake                 bool
	NextSerialNumber           int // The next serial number for the answer
	NextSerialNumberForCorrect int // The next serial number for correct answers
}

type QuizRepository struct {
	DB *sql.DB
}

func (qr *QuizRepository) GetQuestionAndScoringInfo(shortNumber string, currentDateTime time.Time, clientID int64) (*QuestionScoringInfo, error) {
	query := `
		SELECT 
			q.id, 
			q.quiz_id, 
			q.answer, 
			q.score,
			IFNULL((
				SELECT COUNT(*) 
				FROM answers 
				WHERE question_id = q.id AND client_id = ? AND score > 0
			), 0) AS has_scored,
			IFNULL((SELECT COUNT(*) FROM answers WHERE question_id = ? AND client_id = ? AND score = 0
			), 0) AS has_mistake,
			IFNULL((
				SELECT MAX(serial_number) + 1 
				FROM answers 
				WHERE question_id = q.id
			), 1) AS next_serial_number,
			IFNULL((
				SELECT MAX(serial_number_for_correct) + 1 
				FROM answers 
				WHERE question_id = q.id AND score > 0
			), 1) AS next_serial_number_for_correct
		FROM questions q
		JOIN quizzes z ON q.quiz_id = z.id
		JOIN accounts a ON z.account_id = a.id
		WHERE a.short_number = ? AND q.starts_at <= ? AND q.ends_at >= ?
		LIMIT 1
	`

	var result QuestionScoringInfo
	var hasScoredInt, hasMistakeInt int

	err := qr.DB.QueryRow(query, clientID, clientID, shortNumber, currentDateTime, currentDateTime).Scan(
		&result.ID, &result.QuizID, &result.Answer, &result.Score, &hasScoredInt, &hasMistakeInt, &result.NextSerialNumber, &result.NextSerialNumberForCorrect,
	)

	if err != nil {
		return nil, err
	}

	// Convert hasScored integer to boolean
	result.HasScored = hasScoredInt > 0

	return &result, nil
}

func (qr *QuizRepository) InsertAnswer(questionID int64, msg string, dt time.Time, clientID int64, score int, serialNumber int, serialNumberForCorrect int) error {
	_, err := qr.DB.Exec(
		"INSERT INTO answers (question_id, msg, dt, client_id, score, quiz_id, serial_number, serial_number_for_correct) VALUES (?, ?, ?, ?, ?, (SELECT quiz_id FROM questions WHERE id = ?), ?, ?)",
		questionID, msg, dt, clientID, score, questionID, serialNumber, serialNumberForCorrect,
	)
	return err
}

func (qr *QuizRepository) GetIncorrectAnswerCount(questionID, clientID int64) (int, error) {
	var count int
	err := qr.DB.QueryRow("SELECT COUNT(*) FROM answers WHERE question_id = ? AND client_id = ? AND score = 0", questionID, clientID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
