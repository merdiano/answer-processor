package service

import (
	websocket "answers-processor/internal/delivery"
	"answers-processor/internal/domain"
	publisher "answers-processor/internal/infrastructure/rabbitmq/publisher"
	"answers-processor/internal/repository"
	"answers-processor/internal/strategies"
	"answers-processor/pkg/logger"
	"database/sql"

	"time"
)

type Service struct {
	DB          *sql.DB
	LogInstance *logger.Loggers
	strategies  map[string]strategies.ProcessingStrategy
}

const customDateFormat = "2006-01-02T15:04:05"

func NewService(db *sql.DB, publisher publisher.MessagePublisher, wsServer websocket.Broadcaster, logInstance *logger.Loggers) *Service {
	s := &Service{
		DB:          db,
		LogInstance: logInstance,
		strategies:  make(map[string]strategies.ProcessingStrategy),
	}

	// Initialize strategies
	s.strategies["quiz"] = strategies.NewQuizStrategy(publisher, wsServer, db)
	s.strategies["voting"] = strategies.NewVoteStrategy(publisher, wsServer, db)
	s.strategies["shop"] = strategies.NewShopStrategy(publisher, wsServer, db)
	s.strategies["lottery"] = strategies.NewLotteryStrategy(publisher, wsServer, db)
	return s
}

func (s *Service) ProcessMessage(message domain.SMSMessage) {
	if s.DB == nil {
		s.LogInstance.ErrorLogger.Error("Database instance is nil in ProcessMessage")
		return
	}

	parsedDate, err := time.Parse(customDateFormat, message.Date)
	if err != nil {
		s.LogInstance.ErrorLogger.Error("Failed to parse date", "date", message.Date, "error", err)
		return
	}

	clientID, err := repository.InsertClientIfNotExists(s.DB, message.Source)
	if err != nil {
		s.LogInstance.ErrorLogger.Error("Failed to insert or find client", "error", err)
		return
	}

	accountType, err := repository.GetAccountType(s.DB, message.Destination)
	if err != nil {
		s.LogInstance.ErrorLogger.Error("Failed to get account type", "number", message.Destination)
		return
	}

	if strategy, ok := s.strategies[accountType]; ok {
		err = strategy.Process(clientID, message, parsedDate)
		if err != nil {
			s.LogInstance.ErrorLogger.Error(err.Error())
		}
	} else {
		s.LogInstance.ErrorLogger.Error("Unknown account type", "account_type", accountType)
	}
}
