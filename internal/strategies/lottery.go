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

type LotteryStrategy struct {
	publisher   publisher.MessagePublisher
	broadcaster websocket.Broadcaster
	db          *sql.DB
}

func NewLotteryStrategy(publisher publisher.MessagePublisher, broadcaster websocket.Broadcaster, db *sql.DB) ProcessingStrategy {
	return &LotteryStrategy{
		publisher:   publisher,
		broadcaster: broadcaster,
		db:          db,
	}
}

func (ls *LotteryStrategy) Process(clientID int64, message domain.SMSMessage, parsedDate time.Time) error {
	// Implementation of quiz processing logic
	const customDateFormat = "2006-01-02T15:04:05"

	id, code, answer, err := repository.GetLotteryByShortNumber(ls.db, message.Destination, parsedDate)
	if err != nil {
		return fmt.Errorf("Failed to find lot by short number and date: %w", err)
	}

	text := strings.ToLower(strings.TrimSpace(message.Text))
	code = strings.ToLower(strings.TrimSpace(code))

	if text == code {
		//todo insert lottery item bradcast
		starredSrc := utils.StarMiddleDigits(message.Source)

		err = repository.InsertLotteryMessageAndUpdate(ls.db, id, message.Text, parsedDate, clientID)
		if err != nil {
			return fmt.Errorf("Failed to insert lottery message and update: %w", err)
		}

		// Send message notification
		err = ls.publisher.SendMessage(message.Destination, message.Source, answer)
		if err != nil {
			return fmt.Errorf("Failed to send message notification: %w", err)
		}

		// Broadcast to WebSocket
		lotteryMessage := domain.LotteryMessage{
			LotteryID: id,
			Date:      parsedDate.Format(customDateFormat),
			Src:       starredSrc,
		}

		if msg, err := json.MarshalIndent(lotteryMessage, "", "    "); err != nil {
			return fmt.Errorf("Failed to marshal correct answer message: %w", err)
		} else {
			ls.broadcaster.Broadcast(message.Destination, msg)
		}
	}

	return nil
}
