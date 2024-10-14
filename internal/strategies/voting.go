package strategies

import (
	websocket "answers-processor/internal/delivery"
	"answers-processor/internal/domain"
	"answers-processor/internal/infrastructure/rabbitmq/publisher"
	"answers-processor/internal/repository"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type VoteStrategy struct {
	publisher   publisher.MessagePublisher
	broadcaster websocket.Broadcaster
	repo        *repository.VotingRepository
}

func NewVoteStrategy(publisher publisher.MessagePublisher, broadcaster websocket.Broadcaster, repo *repository.VotingRepository) ProcessingStrategy {
	return &VoteStrategy{
		publisher:   publisher,
		broadcaster: broadcaster,
		repo:        repo,
	}
}

func (vs *VoteStrategy) Process(clientID int64, message domain.SMSMessage, parsedDate time.Time) error {
	const customDateFormat = "2006-01-02T15:04:05"
	votingID, status, err := vs.repo.GetVotingDetails(message.Destination, parsedDate)
	if err != nil {
		return fmt.Errorf("Failed to find voting by short number and date: %w", err)
	}

	votingItemID, votingItemTitle, err := vs.repo.GetVotingItemDetails(votingID, message.Text)
	if err != nil {
		return fmt.Errorf("Failed to find voting item by vote code: %w", err)
	}

	hasVoted, err := vs.repo.HasClientVoted(votingID, clientID, status, parsedDate)
	if err != nil {
		return fmt.Errorf("Failed to check if client has voted: %w", err)
	} else if hasVoted {
		log.Printf("client has already Voted")
		return nil
	}

	err = vs.repo.InsertVotingMessageAndUpdateCount(votingID, votingItemID, message.Text, parsedDate, clientID)
	if err != nil {
		return fmt.Errorf("Failed to insert voting message and update count: %w", err)
	}

	smsText := votingItemTitle + " ucin beren sesiniz kabul edildi"
	err = vs.publisher.SendMessage(message.Destination, message.Source, smsText)
	if err != nil {
		log.Printf("Failed to send message notification: %w", err)
	}

	votingMessage := domain.VotingMessage{
		VotingID:     votingID,
		VotingItemID: votingItemID,
		ClientID:     clientID,
		Message:      message.Text,
		Date:         parsedDate.Format(customDateFormat),
	}
	msg, _ := json.MarshalIndent(votingMessage, "", "    ")
	vs.broadcaster.Broadcast(message.Destination, msg)

	return nil

}
