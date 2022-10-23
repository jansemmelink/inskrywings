package main

import "time"

type Ledger struct {
	Transactions []Transaction      `json:"transactions"`
	Accounts     map[string]Account `json:"accountByID"`
}

type Account struct {
	ID string `json:"id" doc:"Unique ID used to pay to this account"`
}

type Transaction struct {
	Timestamp time.Time `json:"timestamp"`
	Amount    Amount    `json:"amount" doc:">0 increase your balance, <0 decrease your balance"`
	Account   *Account  `json:"account"`
	Reference string    `json:"reference"`
	Notes     string    `json:"notes"`
}
