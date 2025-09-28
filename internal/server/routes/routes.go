package routes

import (
	"net/http"

	"github.com/Sush1sui/sniper_bot/internal/server/handlers"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", handlers.IndexHandler)
	return mux
}