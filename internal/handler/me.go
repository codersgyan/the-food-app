package handler

import "net/http"

type addressJSON struct {
	ID      int64  `json:"id"`
	Address string `json:"address"`
}

// GET /v1/me returns the logged-in customer, with their saved addresses.
func (h *handler) me(w http.ResponseWriter, r *http.Request) {
	u, addresses, err := h.svc.Users.Me(r.Context(), claims(r).UserID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	res := struct {
		ID        int64         `json:"id"`
		Name      *string       `json:"name"`
		Phone     string        `json:"phone"`
		Addresses []addressJSON `json:"addresses"`
	}{ID: u.ID, Name: u.Name, Phone: u.Phone, Addresses: []addressJSON{}}
	for _, a := range addresses {
		res.Addresses = append(res.Addresses, addressJSON{ID: a.ID, Address: a.Address})
	}
	writeJSON(w, http.StatusOK, res)
}
