package handler

import "net/http"

// POST /v1/login sends a login code to the phone number.
func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phone string `json:"phone"`
	}
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	if err := h.svc.Auth.StartLogin(r.Context(), req.Phone); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "code sent"})
}

// POST /v1/login/verify checks the code and returns a token for the user.
func (h *handler) verifyLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	token, err := h.svc.Auth.VerifyLogin(r.Context(), req.Phone, req.Code)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}
