package repository

import (
	"time"

	"github.com/ignavan39/ucrm-go/app/models"
)

type MessageRepository interface {
	CreateOneMessage(payload models.MessagePayload, senderId string, deleted bool, status string, createdAt time.Time, chatId string) (*models.Message, error)
}
