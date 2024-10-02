package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type APIServer struct {
	listenAddr string
	store      Storage
}

type APIFunc func(w http.ResponseWriter, r *http.Request) error

type APIError struct {
	Error string `json:"error"`
}

func newApiServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func makeHttpHandler(fn APIFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			writeJson(w, http.StatusBadRequest, APIError{Error: err.Error()}) // handle error
		}
	}
}

func writeJson(w http.ResponseWriter, status int, val any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(val)
}

func (s *APIServer) run() {
	router := mux.NewRouter()
	router.HandleFunc("/deposit", JWTauthMiddleWare(makeHttpHandler(s.handleDoposit), s.store))
	router.HandleFunc("/login", makeHttpHandler(s.handleLogin))
	router.HandleFunc("/account", makeHttpHandler(s.handleAccount))
	router.HandleFunc("/account/{id}", JWTauthMiddleWare(makeHttpHandler(s.handleGetAccountById), s.store))
	router.HandleFunc("/transfer", JWTauthMiddleWare(makeHttpHandler(s.handleTransfer), s.store))

	log.Printf("API server listening on %s", s.listenAddr)
	http.ListenAndServe(s.listenAddr, router)

}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.handleGetAccounts(w)
	}
	if r.Method == "POST" {
		return s.handleCreateAccount(w, r)
	}
	if r.Method == "DELETE" {
		return s.handleDeleteAccount(w, r)
	}
	return fmt.Errorf("methid not allowed %s", r.Method)
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("method not allowed %s", r.Method)
	}

	loginReq := new(LoginRequest)
	if err := json.NewDecoder(r.Body).Decode(loginReq); err != nil {
		return err
	}
	defer r.Body.Close()

	account, err := s.store.GetAccountByNumber(loginReq.AccountNumber)
	fmt.Print(account)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(loginReq.Password)) != nil {
		return fmt.Errorf("ss login credentials")
	}

	token, err := generateJWT(account)
	if err != nil {
		return fmt.Errorf("error generating token: %v", err)
	}

	return writeJson(w, http.StatusOK, map[string]string{"token": token})
}

func (s *APIServer) handleDoposit(w http.ResponseWriter, r *http.Request) error {
	// Initialize a new DepositRequest struct
	depositReq := &DepositRequest{}

	// Decode the request body into depositReq
	if err := json.NewDecoder(r.Body).Decode(depositReq); err != nil {
		return fmt.Errorf("invalid deposit request: %v", err)
	}
	defer r.Body.Close()

	// Ensure the amount is a positive float
	if depositReq.Amount <= 0 {
		return fmt.Errorf("invalid deposit amount")
	}

	// Extract the account from the context
	account := r.Context().Value("account").(*Account)

	// Check if the account number matches
	if account.AccountNumber != depositReq.AccountNumber {
		return fmt.Errorf("unauthorized: You can only deposit into your own account")
	}

	// Log the deposit information
	fmt.Printf("Depositing into account %d, amount is %.2f\n", depositReq.AccountNumber, depositReq.Amount)

	// Update the account balance
	accountToDeposit, err := s.store.GetAccountByNumber(depositReq.AccountNumber)
	if err != nil {
		return fmt.Errorf("account not found: %v", err)
	}

	// Update balance
	accountToDeposit.Balance += depositReq.Amount
	updatedAccount, err := s.store.UpdateAccountBalance(accountToDeposit.AccountNumber, accountToDeposit.Balance)
	if err != nil {
		return fmt.Errorf("error updating account balance: %v", err)
	}

	return writeJson(w, http.StatusOK, updatedAccount)
}

func (s *APIServer) handleGetAccounts(w http.ResponseWriter) error {
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return err
	}
	return writeJson(w, http.StatusOK, accounts)
}

func (s *APIServer) handleGetAccountById(w http.ResponseWriter, r *http.Request) error {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return fmt.Errorf("invalid account id %s", idStr)
	}

	// Extract the account from the context
	account := r.Context().Value("account").(*Account)
	if account.ID != id {
		return fmt.Errorf("unauthorized: You are not allowed to access this account")
	}

	accountData, err := s.store.GetAccountById(id)
	if err != nil {
		return err
	}

	fmt.Printf("Getting account of id : %d", id)

	return writeJson(w, http.StatusOK, accountData)
}

func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	createAccountReq := CreateAccountRequest{}
	if err := json.NewDecoder(r.Body).Decode(&createAccountReq); err != nil {
		return err
	}
	fmt.Printf("Creating account for %s %s", createAccountReq.FirstName, createAccountReq.LastName)

	account, err := NewAccount(createAccountReq.AccountNumber, createAccountReq.FirstName, createAccountReq.LastName, createAccountReq.Password)
	if err != nil {
		fmt.Print("errire1")
		return err
	}
	if err := s.store.CreateAccount(account); err != nil {
		fmt.Print("errire2")
		return err
	}

	return writeJson(w, http.StatusOK, account)
}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) error {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return fmt.Errorf("invalid account id %s", idStr)
	}
	fmt.Printf("Deleting account of id : %d", id)

	err = s.store.DeleteAccount(id)
	if err != nil {
		fmt.Errorf("error deleting account %d  : %s ", id, err)
	}

	return writeJson(w, http.StatusOK, map[string]int{"Deleted account": id})
}

func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) error {
	TransferReq := new(TransferRequest)
	if err := json.NewDecoder(r.Body).Decode(TransferReq); err != nil {
		return err
	}
	defer r.Body.Close()

	return writeJson(w, http.StatusOK, TransferReq)
}

func generateJWT(account *Account) (string, error) {

	claims := &jwt.MapClaims{
		"expiresAt":     time.Now().Add(time.Minute * 15).Unix(),
		"accountnumber": account.AccountNumber,
	}

	mySigning := os.Getenv("JWT_SECRET")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(mySigning))
}

func permissionDenied(w http.ResponseWriter) {
	writeJson(w, http.StatusUnauthorized, APIError{Error: "Permission Denied"})
}

func JWTauthMiddleWare(handlerFunc http.HandlerFunc, s Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Middleware Authenticating request")
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			permissionDenied(w)
			return
		}

		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		token, err := validateJWT(tokenString)
		if err != nil || !token.Valid {
			permissionDenied(w)
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		accountNumber := int64(claims["accountnumber"].(float64))

		account, err := s.GetAccountByNumber(int(accountNumber))
		if err != nil {
			permissionDenied(w)
			return
		}

		ctx := context.WithValue(r.Context(), "account", account)
		r = r.WithContext(ctx)

		handlerFunc(w, r)
	}
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})
}

// func getID(r *http.Request) (int, error) {
// 	idStr := mux.Vars(r)["id"]
// 	id, err := strconv.Atoi(idStr)
// 	if err != nil {
// 		return id, fmt.Errorf("invalid id given %s", idStr)
// 	}
// 	return id, nil
// }

//create login
// add withdraw
// add deposit
