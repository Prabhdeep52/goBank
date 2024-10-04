package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type Storage interface {
	CreateAccount(*Account) error
	DeleteAccount(int) error
	UpdateAccount(*Account) error
	GetAccountById(int) (*Account, error)
	GetAccounts() ([]*Account, error)
	GetAccountByNumber(int) (*Account, error)
	UpdateAccountBalance(int, float64) (*Account, error)
	CreateTransaction(int, int, string, float64) (*Account, error)
}

type PostGresStore struct {
	db *sql.DB
}

func NewPostGresStore() (*PostGresStore, error) {
	fmt.Println("Starting database connection...")
	connStr := "host=localhost port=5432 user=postgres dbname=gobankpostgres password=helloworld sslmode=disable"
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		fmt.Println("Error opening database:", err)
		return nil, err
	}

	fmt.Println("Pinging the database...")
	if err := db.Ping(); err != nil {
		fmt.Println("Error pinging database:", err)
		return nil, err
	}

	fmt.Println("Successfully connected to the database")
	return &PostGresStore{db: db}, nil
}

func (s *PostGresStore) init() error {
	if err := s.createAccountTable(); err != nil {
		return err
	}

	if err := s.createTransactionsTable(); err != nil {
		return err
	}
	return nil
}

func (s *PostGresStore) createAccountTable() error {
	query := `create table if not exists accounts (
		id serial primary key,
		first_name varchar(50),
		last_name varchar(50),
		accountnumber integer unique , 
		balance integer,
		created_at timestamp default current_timestamp,
		password varchar(100)
	)`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostGresStore) createTransactionsTable() error {
	query := `CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    from_account INTEGER NULL,  -- Allow NULL for deposit
    to_account INTEGER NULL,    -- Allow NULL for withdraw
    transactionType VARCHAR(50),
    amount INTEGER,
    transactiontime TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_account) REFERENCES accounts(accountnumber) ON DELETE SET NULL,
    FOREIGN KEY (to_account) REFERENCES accounts(accountnumber) ON DELETE SET NULL,
    CHECK (transactionType IN ('deposit', 'withdraw', 'transfer'))
)`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostGresStore) EnterTransaction() {
	s.db.Close()
}

func (s *PostGresStore) CreateAccount(ac *Account) error {
	query := `insert into accounts (first_name, last_name, accountnumber, balance, created_at, password) 
	values ($1, $2, $3, $4, $5, $6) returning id`

	err := s.db.QueryRow(
		query,
		ac.FirstName,
		ac.LastName,
		ac.AccountNumber,
		ac.Balance,
		ac.CreatedAt,
		ac.Password).Scan(&ac.ID)

	if err != nil {
		return err
	}

	fmt.Println("Account created successfully with ID:", ac.ID)
	return nil
}

func (s *PostGresStore) GetAccountByNumber(accountnumber int) (*Account, error) {
	fmt.Print("Getting account by number called")
	rows, err := s.db.Query("SELECT * FROM accounts WHERE accountnumber = $1", accountnumber)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		return scanAccounts(rows)

	}
	defer rows.Close()
	return nil, fmt.Errorf("Account with number %d not found", accountnumber)
}

func (s *PostGresStore) GetAccounts() ([]*Account, error) {
	rows, err := s.db.Query("SELECT * FROM accounts")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*Account

	for rows.Next() {
		account := new(Account)
		account, err = scanAccounts(rows)
		if err != nil {
			return nil, err

		}
		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return accounts, nil
}

func (s *PostGresStore) GetAccountById(Id int) (*Account, error) {
	rows, err := s.db.Query("SELECT * FROM accounts WHERE id = $1", Id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		return scanAccounts(rows)

	}
	defer rows.Close()
	return nil, fmt.Errorf("Account with id %d not found", Id)
}

func (s *PostGresStore) DeleteAccount(Id int) error {
	_, err := s.db.Exec("DELETE FROM accounts WHERE id = $1", Id)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostGresStore) UpdateAccount(a *Account) error {

	return nil
}

func (s *PostGresStore) UpdateAccountBalance(accountNumber int, newBalance float64) (*Account, error) {
	// Perform the update
	_, err := s.db.Exec("UPDATE accounts SET balance = $1 WHERE accountnumber = $2", newBalance, accountNumber)
	if err != nil {
		return nil, err
	}

	// Fetch the updated account from the database
	updatedAccount := &Account{}
	err = s.db.QueryRow("SELECT id, first_name, last_name, accountnumber, balance, created_at FROM accounts WHERE accountnumber = $1", accountNumber).
		Scan(&updatedAccount.ID, &updatedAccount.FirstName, &updatedAccount.LastName, &updatedAccount.AccountNumber, &updatedAccount.Balance, &updatedAccount.CreatedAt)

	if err != nil {
		return nil, err
	}

	// Return the updated account
	return updatedAccount, nil
}

func (s *PostGresStore) CreateTransaction(fromAccount, toAccount int, transactionType string, amount float64) (*Account, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var query string
	switch transactionType {
	case "transfer":
		query = `INSERT INTO transactions (from_account, to_account, transactionType, amount) 
                 VALUES ($1, $2, $3, $4)`
		_, err = tx.Exec(query, fromAccount, toAccount, transactionType, amount)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(`UPDATE accounts SET balance = balance - $1 WHERE accountnumber = $2`, amount, fromAccount)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(`UPDATE accounts SET balance = balance + $1 WHERE accountnumber = $2`, amount, toAccount)
		if err != nil {
			return nil, err
		}

	case "deposit":
		fmt.Print("Deposit called with amount: for account ", amount, toAccount)
		query = `INSERT INTO transactions (from_account, to_account, transactionType, amount) 
                 VALUES (NULL, $1, $2, $3)` // from_account is NULL for deposits
		_, err = tx.Exec(query, toAccount, transactionType, amount)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(`UPDATE accounts SET balance = balance + $1 WHERE accountnumber = $2`, amount, toAccount)
		if err != nil {
			return nil, err
		}

	case "withdraw":
		query = `INSERT INTO transactions (from_account, to_account, transactionType, amount) 
                 VALUES ($1, NULL, $2, $3)`
		_, err = tx.Exec(query, fromAccount, transactionType, amount)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(`UPDATE accounts SET balance = balance - $1 WHERE accountnumber = $2`, amount, fromAccount)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	updatedAccount := &Account{}
	switch transactionType {
	case "deposit":
		err = s.db.QueryRow("SELECT id, first_name, last_name, accountnumber, balance, created_at FROM accounts WHERE accountnumber = $1", toAccount).
			Scan(&updatedAccount.ID, &updatedAccount.FirstName, &updatedAccount.LastName, &updatedAccount.AccountNumber, &updatedAccount.Balance, &updatedAccount.CreatedAt)
	case "withdraw", "transfer":
		err = s.db.QueryRow("SELECT id, first_name, last_name, accountnumber, balance, created_at FROM accounts WHERE accountnumber = $1", fromAccount).
			Scan(&updatedAccount.ID, &updatedAccount.FirstName, &updatedAccount.LastName, &updatedAccount.AccountNumber, &updatedAccount.Balance, &updatedAccount.CreatedAt)
	}
	if err != nil {
		return nil, err
	}

	return updatedAccount, nil
}

func scanAccounts(rows *sql.Rows) (*Account, error) {
	account := new(Account)
	if err := rows.Scan(
		&account.ID,
		&account.FirstName,
		&account.LastName,
		&account.AccountNumber,
		&account.Balance,
		&account.CreatedAt,
		&account.Password,
	); err != nil {
		return account, err
	}
	return account, nil
}
