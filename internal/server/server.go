package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/stockyard-dev/stockyard-ledger2/internal/store"
)

type Server struct { db *store.DB; mux *http.ServeMux }

func New(db *store.DB, limits Limits) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), limits: limits}
	s.mux.HandleFunc("GET /api/accounts", s.listAccounts)
	s.mux.HandleFunc("POST /api/accounts", s.createAccount)
	s.mux.HandleFunc("GET /api/accounts/{id}", s.getAccount)
	s.mux.HandleFunc("DELETE /api/accounts/{id}", s.deleteAccount)

	s.mux.HandleFunc("GET /api/transactions", s.listTransactions)
	s.mux.HandleFunc("POST /api/transactions", s.createTransaction)
	s.mux.HandleFunc("DELETE /api/transactions/{id}", s.deleteTransaction)

	s.mux.HandleFunc("GET /api/budgets", s.listBudgets)
	s.mux.HandleFunc("POST /api/budgets", s.setBudget)
	s.mux.HandleFunc("DELETE /api/budgets/{id}", s.deleteBudget)

	s.mux.HandleFunc("GET /api/summary", s.monthSummary)
	s.mux.HandleFunc("GET /api/breakdown", s.categoryBreakdown)
	s.mux.HandleFunc("GET /api/categories", s.categories)
	s.mux.HandleFunc("GET /api/networth", s.netWorth)
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)

	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)
s.mux.HandleFunc("GET /api/tier",func(w http.ResponseWriter,r *http.Request){writeJSON(w,200,map[string]any{"tier":s.limits.Tier,"upgrade_url":"https://stockyard.dev/ledger2/"})})
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }
func writeJSON(w http.ResponseWriter, code int, v any) { w.Header().Set("Content-Type","application/json"); w.WriteHeader(code); json.NewEncoder(w).Encode(v) }
func writeErr(w http.ResponseWriter, code int, msg string) { writeJSON(w, code, map[string]string{"error": msg}) }
func (s *Server) root(w http.ResponseWriter, r *http.Request) { if r.URL.Path != "/" { http.NotFound(w, r); return }; http.Redirect(w, r, "/ui", http.StatusFound) }

func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"accounts": orEmpty(s.db.ListAccounts())}) }
func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var a store.Account; json.NewDecoder(r.Body).Decode(&a)
	if a.Name == "" { writeErr(w, 400, "name required"); return }
	s.db.CreateAccount(&a); writeJSON(w, 201, s.db.GetAccount(a.ID))
}
func (s *Server) getAccount(w http.ResponseWriter, r *http.Request) {
	a := s.db.GetAccount(r.PathValue("id")); if a == nil { writeErr(w, 404, "not found"); return }; writeJSON(w, 200, a)
}
func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) { s.db.DeleteAccount(r.PathValue("id")); writeJSON(w, 200, map[string]string{"deleted":"ok"}) }

func (s *Server) listTransactions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	writeJSON(w, 200, map[string]any{"transactions": orEmpty(s.db.ListTransactions(q.Get("account_id"), q.Get("month"), q.Get("category"), 200))})
}
func (s *Server) createTransaction(w http.ResponseWriter, r *http.Request) {
	var t store.Transaction; json.NewDecoder(r.Body).Decode(&t)
	if t.AccountID == "" { writeErr(w, 400, "account_id required"); return }
	if t.Amount == 0 { writeErr(w, 400, "amount required"); return }
	s.db.CreateTransaction(&t); writeJSON(w, 201, t)
}
func (s *Server) deleteTransaction(w http.ResponseWriter, r *http.Request) { s.db.DeleteTransaction(r.PathValue("id")); writeJSON(w, 200, map[string]string{"deleted":"ok"}) }

func (s *Server) listBudgets(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"budgets": orEmpty(s.db.ListBudgets(r.URL.Query().Get("month")))}) }
func (s *Server) setBudget(w http.ResponseWriter, r *http.Request) {
	var req struct{ Category string `json:"category"`; Month string `json:"month"`; Amount float64 `json:"amount"` }
	json.NewDecoder(r.Body).Decode(&req)
	if req.Category == "" || req.Amount <= 0 { writeErr(w, 400, "category and amount required"); return }
	s.db.SetBudget(req.Category, req.Month, req.Amount); writeJSON(w, 200, map[string]string{"set":"ok"})
}
func (s *Server) deleteBudget(w http.ResponseWriter, r *http.Request) { s.db.DeleteBudget(r.PathValue("id")); writeJSON(w, 200, map[string]string{"deleted":"ok"}) }

func (s *Server) monthSummary(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.db.MonthSummary(r.URL.Query().Get("month"))) }
func (s *Server) categoryBreakdown(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"breakdown": orEmpty(s.db.CategoryBreakdown(r.URL.Query().Get("month")))}) }
func (s *Server) categories(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"categories": orEmpty(s.db.Categories())}) }
func (s *Server) netWorth(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]float64{"net_worth": s.db.NetWorth()}) }
func (s *Server) stats(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.db.Stats()) }
func (s *Server) health(w http.ResponseWriter, r *http.Request) { st := s.db.Stats(); writeJSON(w, 200, map[string]any{"status":"ok","service":"ledger2","accounts":st.Accounts,"net_worth":st.NetWorth}) }

func orEmpty[T any](s []T) []T { if s == nil { return []T{} }; return s }
func init() { log.SetFlags(log.LstdFlags | log.Lshortfile) }
