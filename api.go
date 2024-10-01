package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
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

	router.HandleFunc("/login", makeHttpHandler(s.handleLogin))
	router.HandleFunc("/account", makeHttpHandler(s.handleAccount))
	router.HandleFunc("/account/{id}", JWTauthMiddleWare(makeHttpHandler(s.handleGetAccountById), s.store))
	router.HandleFunc("/transfer", makeHttpHandler(s.handleTransfer))

	log.Printf("API server listening on %s", s.listenAddr)
	http.ListenAndServe(s.listenAddr, router)

}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.handleGetAccounts(w, r)
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
		return fmt.Errorf("Method not allowed %s", r.Method)
	}
	loginReq := new(LoginRequest)
	if err := json.NewDecoder(r.Body).Decode(loginReq); err != nil {
		return err
	}
	defer r.Body.Close()
	return writeJson(w, http.StatusOK, loginReq)
}

func (s *APIServer) handleGetAccounts(w http.ResponseWriter, r *http.Request) error {
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return err
	}
	return writeJson(w, http.StatusOK, accounts)
}

func (s *APIServer) handleGetAccountById(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		idStr := mux.Vars(r)["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return fmt.Errorf("invalid account id %s", idStr)
		}
		account, err := s.store.GetAccountById(id)
		if err != nil {
			return err
		}

		fmt.Printf("Getting account of id : %d", id)

		return writeJson(w, http.StatusOK, account)
	}
	if r.Method == "DELETE" {
		return s.handleDeleteAccount(w, r)
	}
	fmt.Errorf("Method not allowed %s", r.Method)
	return writeJson(w, http.StatusBadRequest, APIError{Error: "Method not allowed"})
}

func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	createAccountReq := CreateAccountRequest{}
	if err := json.NewDecoder(r.Body).Decode(&createAccountReq); err != nil {
		return err
	}
	fmt.Printf("Creating account for %s %s", createAccountReq.FirstName, createAccountReq.LastName)

	account, err := NewAccount(createAccountReq.FirstName, createAccountReq.LastName, createAccountReq.Password)
	if err != nil {
		return err
	}
	if err := s.store.CreateAccount(account); err != nil {
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
		fmt.Errorf("Error deleting account %d  : %s ", id, err)
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
		"accountnumber": account.Number,
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
		fmt.Printf("middleware Authenticating request")
		tokenString := r.Header.Get("x-jwt-token")
		fmt.Println(tokenString)

		token, err := validateJWT(tokenString)
		if err != nil {
			permissionDenied(w)
			return
		}

		if !token.Valid {
			permissionDenied(w)
			return
		}

		userID, err := getID(r)
		if err != nil {
			permissionDenied(w)
			return
		}

		fmt.Println("User ID", userID)

		account, err := s.GetAccountById(userID)
		if err != nil {
			permissionDenied(w)
			return
		}

		claims := token.Claims.(jwt.MapClaims)

		if account.Number != int64(claims["accountnumber"].(float64)) {
			permissionDenied(w)
			return
		}

		handlerFunc(w, r)
	}
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})
}

func getID(r *http.Request) (int, error) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return id, fmt.Errorf("invalid id given %s", idStr)
	}
	return id, nil
}

//create login
// add withdraw
// add deposit
