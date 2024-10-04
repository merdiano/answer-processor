package strategies

import (
	websocket "answers-processor/internal/delivery"
	"answers-processor/internal/domain"
	"answers-processor/internal/infrastructure/rabbitmq/publisher"
	"answers-processor/internal/repository"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type ShopStrategy struct {
	publisher   publisher.MessagePublisher
	broadcaster websocket.Broadcaster
	db          *sql.DB
}

func NewShopStrategy(publisher publisher.MessagePublisher, broadcaster websocket.Broadcaster, db *sql.DB) ProcessingStrategy {
	return &ShopStrategy{
		publisher:   publisher,
		broadcaster: broadcaster,
		db:          db,
	}
}

func (ss *ShopStrategy) Process(clientID int64, message domain.SMSMessage, parsedDate time.Time) error {
	const customDateFormat = "2006-01-02T15:04:05"
	lotID, description, err := repository.GetLotDetailsByShortNumber(ss.db, message.Destination, parsedDate)
	if err != nil {
		return fmt.Errorf("Failed to find lot by short number and date: %w", err)
	}

	err = repository.InsertLotMessageAndUpdate(ss.db, lotID, message.Text, parsedDate, clientID)
	if err != nil {
		return fmt.Errorf("Failed to insert lot SMS message and update: %w", err)
	}

	// Send message notification
	err = ss.publisher.SendMessage(message.Destination, message.Source, description)
	if err != nil {
		return fmt.Errorf("Failed to send message notification: %w", err)
	}

	// Broadcast to WebSocket
	shoppingMessage := domain.ShoppingMessage{
		LotID:    lotID,
		ClientID: clientID,
		Message:  message.Text,
		Date:     parsedDate.Format(customDateFormat),
		Src:      message.Source,
	}
	msg, _ := json.MarshalIndent(shoppingMessage, "", "    ")
	ss.broadcaster.Broadcast(message.Destination, msg)
	return nil
}
