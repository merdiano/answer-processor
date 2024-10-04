package strategies

import (
	websocket "answers-processor/internal/delivery"
	"answers-processor/internal/domain"
	"answers-processor/internal/infrastructure/rabbitmq/publisher"
	"answers-processor/internal/repository"
	"answers-processor/pkg/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type QuizStrategy struct {
	publisher   publisher.MessagePublisher
	broadcaster websocket.Broadcaster
	db          *sql.DB
}

func NewQuizStrategy(publisher publisher.MessagePublisher, broadcaster websocket.Broadcaster, db *sql.DB) ProcessingStrategy {
	return &QuizStrategy{
		publisher:   publisher,
		broadcaster: broadcaster,
		db:          db,
	}
}

func (qs *QuizStrategy) Process(clientID int64, message domain.SMSMessage, parsedDate time.Time) error {

	text := message.Text

	// _, questions, questionIDs, quizID, err := repository.GetAccountAndQuestions(qs.db, destination, parsedDate)
	questionInfo, err := repository.GetQuestionAndScoringInfo(qs.db, message.Destination, parsedDate, clientID)
	if err != nil {
		return fmt.Errorf("Failed to find quiz and questions: %w", err)
	}

	isCorrect := compareAnswers(questionInfo.Answer, text)
	const customDateFormat = "2006-01-02T15:04:05"
	if isCorrect && !questionInfo.HasScored {

		err = repository.InsertAnswer(qs.db, questionInfo.ID, text, parsedDate, clientID, questionInfo.Score, questionInfo.NextSerialNumber, questionInfo.NextSerialNumberForCorrect)
		if err != nil {
			return fmt.Errorf("Failed to insert answer: %w", err)
		}

		starredSrc := utils.StarMiddleDigits(message.Source)

		correctAnswerMessage := domain.CorrectAnswerMessage{
			Answer:                 text,
			Score:                  questionInfo.Score,
			Date:                   parsedDate.Format(customDateFormat),
			SerialNumber:           questionInfo.NextSerialNumber,
			SerialNumberForCorrect: questionInfo.NextSerialNumberForCorrect,
			StarredSrc:             starredSrc,
			QuizID:                 questionInfo.QuizID,
			QuestionID:             questionInfo.ID,
		}

		if msg, err := json.MarshalIndent(correctAnswerMessage, "", "    "); err != nil {
			return fmt.Errorf("Failed to marshal correct answer message: %w", err)
		} else {
			qs.broadcaster.Broadcast(message.Destination, msg)
		}

	} else {
		incorrectAnswerCount, err := repository.GetIncorrectAnswerCount(qs.db, questionInfo.ID, clientID)
		if err != nil {
			//qs.service.LogInstance.ErrorLogger.Error("Failed to get incorrect answer count", "error", err)
			return fmt.Errorf("Failed to get incorrect answer count: %w", err)
		}

		if incorrectAnswerCount == 0 {
			err = repository.InsertAnswer(qs.db, questionInfo.ID, text, parsedDate, clientID, 0, questionInfo.NextSerialNumber, questionInfo.NextSerialNumberForCorrect)
			if err != nil {
				return fmt.Errorf("Failed to insert answer: %w", err)
			}
		}
	}

	return nil
}
func compareAnswers(correctAnswersText string, userAnswer string) bool {
	userAnswer = strings.ToLower(strings.TrimSpace(userAnswer))
	correctAnswers := strings.Split(correctAnswersText, ",")

	for _, correctAnswer := range correctAnswers {
		correctAnswer = strings.ToLower(strings.TrimSpace(correctAnswer))

		if correctAnswer == userAnswer {
			return true
		}
	}
	return false
}
