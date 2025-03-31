package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"wallet-service/internal/models"
	"wallet-service/internal/repository"
	"wallet-service/internal/service"

	"github.com/google/uuid"
)

type WalletHandler struct {
	service *service.WalletService
}

func NewWalletHandler(service *service.WalletService) *WalletHandler {
	return &WalletHandler{
		service: service,
	}
}

func (h *WalletHandler) CreateWallet(w http.ResponseWriter, r *http.Request) {
	wallet, err := h.service.CreateWallet(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondWithJSON(w, http.StatusCreated, wallet)
}

func (h *WalletHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.URL.Path, "/")
	if len(path) < 5 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	walletID, err := uuid.Parse(path[4])
	if err != nil {
		http.Error(w, "Invalid wallet ID", http.StatusBadRequest)
		return
	}

	wallet, err := h.service.GetWallet(r.Context(), walletID)
	if err != nil {
		if errors.Is(err, repository.ErrWalletNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondWithJSON(w, http.StatusOK, wallet)
}

func (h *WalletHandler) ProcessOperation(w http.ResponseWriter, r *http.Request) {
	var operation models.WalletOperation
	if err := json.NewDecoder(r.Body).Decode(&operation); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	wallet, err := h.service.ProcessOperation(r.Context(), operation)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrWalletNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, repository.ErrInsufficientFunds):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	respondWithJSON(w, http.StatusOK, wallet)
}

func respondWithJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
