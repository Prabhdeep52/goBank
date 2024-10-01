package main

import (
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Number   int64  `json:"number"`
	Password string `json:"password"`
}

type CreateAccountRequest struct {
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Password  string `json:"password"`
}

type TransferRequest struct {
	transerTo int `json:"transferTo"`
	Amount    int `json:"amount"`
}

type Account struct {
	ID          int       `json:"id"`
	FirstName   string    `json:"firstname"`
	LastName    string    `json:"lastname"`
	EncryptPass string    `json:"-"`
	Number      int64     `json:"number"`
	Balance     float64   `json:"balance"`
	CreatedAt   time.Time `json:"createdAt"`
	Password    string    `json:"-"`
}

func NewAccount(firstName, LastName, password string) (*Account, error) {
	encPw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &Account{
		FirstName:   firstName,
		LastName:    LastName,
		Number:      int64(rand.Intn(1000000)),
		EncryptPass: string(encPw),
		Balance:     0.0,
		CreatedAt:   time.Now().UTC(),
	}, nil
}
