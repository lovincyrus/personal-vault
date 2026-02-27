package api

import (
	_ "embed"
	"net/http"
)

//go:embed ui/onboarding.html
var onboardingHTML []byte

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(onboardingHTML)
}
