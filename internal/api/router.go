package api

import (
	"net/http"
	"wallet-service/internal/service"
)

func NewRouter(walletService *service.WalletService) *http.ServeMux {
	handler := NewWalletHandler(walletService)
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/wallets", handler.CreateWallet)
	mux.HandleFunc("GET /api/v1/wallets/{id}", handler.GetWallet)
	mux.HandleFunc("POST /api/vi/wallet", handler.ProcessOperation)
	return mux
}
