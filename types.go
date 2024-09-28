package main

import (
	"math/rand"
	"time"
)

type createAccountRequest struct {
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
}

type Account struct {
	ID        int       `json:"id"`
	FirstName string    `json:"firstname"`
	LastName  string    `json:"lastname"`
	Number    int64     `json:"number"`
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewAccount(firstName, LastName string) *Account {
	return &Account{
		FirstName: firstName,
		LastName:  LastName,
		Number:    int64(rand.Intn(1000000)),
		Balance:   0.0,
		CreatedAt: time.Now().UTC(),
	}
}
