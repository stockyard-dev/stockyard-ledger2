package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ db *sql.DB }

type Account struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"` // checking, savings, credit, cash, investment
	Currency  string  `json:"currency"`
	CreatedAt string  `json:"created_at"`
	Balance   float64 `json:"balance"`
}

type Transaction struct {
	ID          string  `json:"id"`
	AccountID   string  `json:"account_id"`
	AccountName string  `json:"account_name,omitempty"`
	Date        string  `json:"date"`
	Payee       string  `json:"payee"`
	Category    string  `json:"category,omitempty"`
	Amount      float64 `json:"amount"` // positive=income, negative=expense
	Note        string  `json:"note,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

type Budget struct {
	ID       string  `json:"id"`
	Category string  `json:"category"`
	Month    string  `json:"month"` // YYYY-MM
	Amount   float64 `json:"amount"`
	Spent    float64 `json:"spent"`
	Remaining float64 `json:"remaining"`
}

type CategorySummary struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
	Count    int     `json:"count"`
}

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil { return nil, err }
	dsn := filepath.Join(dataDir, "ledger2.db") + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil { return nil, err }
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS accounts (id TEXT PRIMARY KEY, name TEXT NOT NULL, type TEXT DEFAULT 'checking', currency TEXT DEFAULT 'USD', created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS transactions (id TEXT PRIMARY KEY, account_id TEXT NOT NULL REFERENCES accounts(id), date TEXT NOT NULL, payee TEXT DEFAULT '', category TEXT DEFAULT '', amount REAL NOT NULL, note TEXT DEFAULT '', created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS budgets (id TEXT PRIMARY KEY, category TEXT NOT NULL, month TEXT NOT NULL, amount REAL NOT NULL, UNIQUE(category, month))`,
		`CREATE INDEX IF NOT EXISTS idx_txn_account ON transactions(account_id)`,
		`CREATE INDEX IF NOT EXISTS idx_txn_date ON transactions(date)`,
		`CREATE INDEX IF NOT EXISTS idx_txn_category ON transactions(category)`,
	} {
		if _, err := db.Exec(q); err != nil { return nil, fmt.Errorf("migrate: %w", err) }
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }
func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string { return time.Now().UTC().Format(time.RFC3339) }
func thisMonth() string { return time.Now().Format("2006-01") }

// ── Accounts ──

func (d *DB) CreateAccount(a *Account) error {
	a.ID = genID(); a.CreatedAt = now()
	if a.Type == "" { a.Type = "checking" }
	if a.Currency == "" { a.Currency = "USD" }
	_, err := d.db.Exec(`INSERT INTO accounts (id,name,type,currency,created_at) VALUES (?,?,?,?,?)`,
		a.ID, a.Name, a.Type, a.Currency, a.CreatedAt)
	return err
}

func (d *DB) calcBalance(acctID string) float64 {
	var bal float64
	d.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM transactions WHERE account_id=?`, acctID).Scan(&bal)
	return bal
}

func (d *DB) GetAccount(id string) *Account {
	var a Account
	if err := d.db.QueryRow(`SELECT id,name,type,currency,created_at FROM accounts WHERE id=?`, id).Scan(&a.ID, &a.Name, &a.Type, &a.Currency, &a.CreatedAt); err != nil { return nil }
	a.Balance = d.calcBalance(a.ID)
	return &a
}

func (d *DB) ListAccounts() []Account {
	rows, _ := d.db.Query(`SELECT id,name,type,currency,created_at FROM accounts ORDER BY type, name`)
	if rows == nil { return nil }; defer rows.Close()
	var out []Account
	for rows.Next() {
		var a Account; rows.Scan(&a.ID, &a.Name, &a.Type, &a.Currency, &a.CreatedAt)
		a.Balance = d.calcBalance(a.ID)
		out = append(out, a)
	}
	return out
}

func (d *DB) DeleteAccount(id string) error {
	d.db.Exec(`DELETE FROM transactions WHERE account_id=?`, id)
	_, err := d.db.Exec(`DELETE FROM accounts WHERE id=?`, id)
	return err
}

// ── Transactions ──

func (d *DB) CreateTransaction(t *Transaction) error {
	t.ID = genID(); t.CreatedAt = now()
	if t.Date == "" { t.Date = time.Now().Format("2006-01-02") }
	_, err := d.db.Exec(`INSERT INTO transactions (id,account_id,date,payee,category,amount,note,created_at) VALUES (?,?,?,?,?,?,?,?)`,
		t.ID, t.AccountID, t.Date, t.Payee, t.Category, t.Amount, t.Note, t.CreatedAt)
	return err
}

func (d *DB) ListTransactions(accountID, month, category string, limit int) []Transaction {
	if limit <= 0 { limit = 100 }
	where := []string{"1=1"}; args := []any{}
	if accountID != "" { where = append(where, "t.account_id=?"); args = append(args, accountID) }
	if month != "" { where = append(where, "t.date LIKE ?"); args = append(args, month+"%") }
	if category != "" { where = append(where, "t.category=?"); args = append(args, category) }
	w := strings.Join(where, " AND ")
	rows, _ := d.db.Query(`SELECT t.id,t.account_id,t.date,t.payee,t.category,t.amount,t.note,t.created_at,COALESCE(a.name,'') FROM transactions t LEFT JOIN accounts a ON t.account_id=a.id WHERE `+w+` ORDER BY t.date DESC, t.created_at DESC LIMIT ?`, append(args, limit)...)
	if rows == nil { return nil }; defer rows.Close()
	var out []Transaction
	for rows.Next() {
		var t Transaction
		rows.Scan(&t.ID, &t.AccountID, &t.Date, &t.Payee, &t.Category, &t.Amount, &t.Note, &t.CreatedAt, &t.AccountName)
		out = append(out, t)
	}
	return out
}

func (d *DB) DeleteTransaction(id string) error { _, err := d.db.Exec(`DELETE FROM transactions WHERE id=?`, id); return err }

// ── Budgets ──

func (d *DB) SetBudget(category, month string, amount float64) error {
	id := genID()
	_, err := d.db.Exec(`INSERT OR REPLACE INTO budgets (id,category,month,amount) VALUES (?,?,?,?)`, id, category, month, amount)
	return err
}

func (d *DB) ListBudgets(month string) []Budget {
	if month == "" { month = thisMonth() }
	rows, _ := d.db.Query(`SELECT id,category,month,amount FROM budgets WHERE month=? ORDER BY category`, month)
	if rows == nil { return nil }; defer rows.Close()
	var out []Budget
	for rows.Next() {
		var b Budget; rows.Scan(&b.ID, &b.Category, &b.Month, &b.Amount)
		d.db.QueryRow(`SELECT COALESCE(SUM(ABS(amount)),0) FROM transactions WHERE category=? AND amount<0 AND date LIKE ?`, b.Category, month+"%").Scan(&b.Spent)
		b.Remaining = b.Amount - b.Spent
		out = append(out, b)
	}
	return out
}

func (d *DB) DeleteBudget(id string) error { _, err := d.db.Exec(`DELETE FROM budgets WHERE id=?`, id); return err }

// ── Summaries ──

func (d *DB) MonthSummary(month string) map[string]float64 {
	if month == "" { month = thisMonth() }
	var income, expenses float64
	d.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM transactions WHERE amount>0 AND date LIKE ?`, month+"%").Scan(&income)
	d.db.QueryRow(`SELECT COALESCE(SUM(ABS(amount)),0) FROM transactions WHERE amount<0 AND date LIKE ?`, month+"%").Scan(&expenses)
	return map[string]float64{"income": income, "expenses": expenses, "net": income - expenses}
}

func (d *DB) CategoryBreakdown(month string) []CategorySummary {
	if month == "" { month = thisMonth() }
	rows, _ := d.db.Query(`SELECT COALESCE(NULLIF(category,''),'uncategorized'), SUM(ABS(amount)), COUNT(*) FROM transactions WHERE amount<0 AND date LIKE ? GROUP BY category ORDER BY SUM(ABS(amount)) DESC`, month+"%")
	if rows == nil { return nil }; defer rows.Close()
	var out []CategorySummary
	for rows.Next() {
		var c CategorySummary; rows.Scan(&c.Category, &c.Total, &c.Count)
		out = append(out, c)
	}
	return out
}

func (d *DB) NetWorth() float64 {
	var total float64
	d.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM transactions`).Scan(&total)
	return total
}

func (d *DB) Categories() []string {
	rows, _ := d.db.Query(`SELECT DISTINCT category FROM transactions WHERE category!='' ORDER BY category`)
	if rows == nil { return nil }; defer rows.Close()
	var out []string
	for rows.Next() { var c string; rows.Scan(&c); out = append(out, c) }
	return out
}

type Stats struct {
	Accounts     int     `json:"accounts"`
	Transactions int     `json:"transactions"`
	NetWorth     float64 `json:"net_worth"`
	MonthIncome  float64 `json:"month_income"`
	MonthExpenses float64 `json:"month_expenses"`
}

func (d *DB) Stats() Stats {
	var s Stats
	d.db.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&s.Accounts)
	d.db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&s.Transactions)
	s.NetWorth = d.NetWorth()
	m := d.MonthSummary("")
	s.MonthIncome = m["income"]; s.MonthExpenses = m["expenses"]
	return s
}
