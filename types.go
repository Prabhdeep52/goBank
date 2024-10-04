package main

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	AccountNumber int    `json:"accountnumber"`
	Password      string `json:"password"`
}

type DepositRequest struct {
	AccountNumber int     `json:"accountnumber"`
	Amount        float64 `json:"amount"`
}

type WithdrawRequest struct {
	AccountNumber int     `json:"accountnumber"`
	Amount        float64 `json:"amount"`
}

type CreateAccountRequest struct {
	AccountNumber int    `json:"accountnumber"`
	FirstName     string `json:"firstname"`
	LastName      string `json:"lastname"`
	Password      string `json:"password"`
}

type TransferRequest struct {
	FromAccountNumber int     `json:"fromAccountNumber"`
	ToAccountNumber   int     `json:"toAccountNumber"`
	Amount            float64 `json:"amount"`
}

type Account struct {
	ID            int       `json:"id"`
	FirstName     string    `json:"firstname"`
	LastName      string    `json:"lastname"`
	AccountNumber int       `json:"accountnumber"`
	Balance       float64   `json:"balance"`
	CreatedAt     time.Time `json:"createdAt"`
	Password      string    `json:"password"`
}

func NewAccount(accountnumber int, firstName, LastName, password string) (*Account, error) {
	encPw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &Account{
		FirstName:     firstName,
		LastName:      LastName,
		AccountNumber: accountnumber,
		Balance:       0.0,
		CreatedAt:     time.Now().UTC(),
		Password:      string(encPw),
	}, nil
}
