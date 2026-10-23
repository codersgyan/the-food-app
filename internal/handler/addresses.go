package handler

import "net/http"

// POST /v1/addresses saves a delivery address for the logged-in customer.
func (h *handler) addAddress(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Address string `json:"address"`
	}
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	id, err := h.svc.Users.AddAddress(r.Context(), claims(r).UserID, req.Address)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}
