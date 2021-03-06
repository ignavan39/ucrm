package user

import "ucrm/app/models"

type Repository interface {
	GetOneByEmail(email string, password string) (*models.User, error)
	Create(email string, password string) (*models.User, error)
	UpdatePassword(email string, password string) (*models.User, error)
}
