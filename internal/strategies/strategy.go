package strategies

import (
	"answers-processor/internal/domain"
	"time"
)

type ProcessingStrategy interface {
	Process(clientID int64, message domain.SMSMessage, parsedDate time.Time) error
}
