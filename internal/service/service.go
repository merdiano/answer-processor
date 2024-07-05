package service

import (
	"answers-processor/internal/repository"
	"answers-processor/pkg/logger"
	"database/sql"
	"strings"
	"time"
)

const customDateFormat = "2006-01-02T15:04:05"

func ProcessMessage(db *sql.DB, body []byte, logInstance *logger.Loggers) {
	message := string(body)
	parts := parseMessageParts(message)
	if len(parts) < 4 { // Ensure there are at least 4 parts: src, dst, txt, date
		logInstance.ErrorLogger.Error("Failed to parse message: input does not match format", "message", message)
		return
	}

	source := parts["src"]
	destination := parts["dst"]
	text := parts["txt"]
	date := parts["date"]

	parsedDate, err := time.Parse(customDateFormat, date)
	if err != nil {
		logInstance.ErrorLogger.Error("Failed to parse date", "date", date, "error", err)
		return
	}

	clientID, err := repository.InsertClientIfNotExists(db, source)
	if err != nil {
		logInstance.ErrorLogger.Error("Failed to insert or find client", "error", err)
		return
	}

	accountType, err := repository.GetAccountType(db, destination)
	if err != nil {
		logInstance.ErrorLogger.Error("Failed to get account type", "error", err)
		return
	}

	switch accountType {
	case "quiz":
		processQuiz(db, clientID, destination, text, parsedDate, logInstance)
	case "voting":
		processVoting(clientID, destination, parsedDate, logInstance)
	default:
		logInstance.ErrorLogger.Error("Unknown account type", "account_type", accountType)
	}
}

func processQuiz(db *sql.DB, clientID int64, destination, text string, parsedDate time.Time, logInstance *logger.Loggers) {
	title, questions, questionIDs, err := repository.GetAccountAndQuestions(db, destination, parsedDate)
	if err != nil {
		logInstance.ErrorLogger.Error("Failed to find quiz and questions by short number and date", "error", err)
		return
	}

	logInstance.InfoLogger.Info("Quiz found", "title", title)
	for i, question := range questions {
		logInstance.InfoLogger.Info("Question found", "question", question)
		correctAnswers, err := repository.GetQuestionAnswers(db, questionIDs[i])
		if err != nil {
			logInstance.ErrorLogger.Error("Failed to get question answers", "error", err)
			continue
		}

		isCorrect := compareAnswers(correctAnswers, text)
		score := 0
		if isCorrect {
			score, err = repository.GetQuestionScore(db, questionIDs[i])
			if err != nil {
				logInstance.ErrorLogger.Error("Failed to get question score", "error", err)
				continue
			}
		}

		if repository.HasClientScored(db, questionIDs[i], clientID) {
			logInstance.InfoLogger.Info("Client has already answered with a score", "client_id", clientID, "question_id", questionIDs[i])
			continue
		}

		serialNumber, err := repository.GetNextSerialNumber(db, questionIDs[i])
		if err != nil {
			logInstance.ErrorLogger.Error("Failed to get next serial number", "error", err)
			continue
		}

		serialNumberForCorrect := 0
		if isCorrect {
			serialNumberForCorrect, err = repository.GetNextSerialNumberForCorrect(db, questionIDs[i])
			if err != nil {
				logInstance.ErrorLogger.Error("Failed to get next serial number for correct answers", "error", err)
				continue
			}
		}

		err = repository.InsertAnswer(db, questionIDs[i], text, parsedDate, clientID, score, serialNumber, serialNumberForCorrect)
		if err != nil {
			logInstance.ErrorLogger.Error("Failed to insert answer", "error", err)
			continue
		}
		logInstance.InfoLogger.Info("Answer inserted", "question_id", questionIDs[i], "is_correct", isCorrect, "score", score, "serial_number", serialNumber, "serial_number_for_correct", serialNumberForCorrect)
	}
}

func processVoting(clientID int64, destination string, parsedDate time.Time, logInstance *logger.Loggers) {
	logInstance.InfoLogger.Info("Processing voting logic", "client_id", clientID, "destination", destination, "date", parsedDate)
	// Placeholder for voting logic
	// Implement the actual voting logic here
}

func compareAnswers(correctAnswers []string, userAnswer string) bool {
	userAnswer = sanitizeAnswer(userAnswer)
	for _, correctAnswer := range correctAnswers {
		if sanitizeAnswer(correctAnswer) == userAnswer {
			return true
		}
	}
	return false
}

func sanitizeAnswer(answer string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(answer), " ", ""))
}

func parseMessageParts(message string) map[string]string {
	result := make(map[string]string)
	parts := strings.Split(message, ", ")
	for _, part := range parts {
		if strings.Contains(part, "=") {
			keyValue := strings.SplitN(part, "=", 2)
			if len(keyValue) == 2 {
				key := keyValue[0]
				value := keyValue[1]
				result[key] = value
			}
		}
	}
	return result
}
