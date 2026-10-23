package handler

import (
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/codersgyan/the-food-app/internal/service"
)

// GET /v1/restaurants?near=12.9352,77.6245 lists the restaurants that deliver
// to that spot on the map, nearest first. The phone sends its own location.
func (h *handler) restaurantsNear(w http.ResponseWriter, r *http.Request) {
	lat, lng, ok := parseLatLng(r.URL.Query().Get("near"))
	if !ok {
		writeError(w, r, &service.Error{Kind: service.ErrInvalid, Msg: "near must be lat,lng, like near=12.9352,77.6245"})
		return
	}
	list, err := h.svc.Restaurants.Near(r.Context(), lat, lng)
	if err != nil {
		writeError(w, r, err)
		return
	}

	type restaurantJSON struct {
		ID         int64   `json:"id"`
		Name       string  `json:"name"`
		Address    string  `json:"address"`
		DistanceKm float64 `json:"distance_km"`
	}
	res := struct {
		Restaurants []restaurantJSON `json:"restaurants"`
	}{Restaurants: []restaurantJSON{}}
	for _, rest := range list {
		res.Restaurants = append(res.Restaurants, restaurantJSON{
			ID:         rest.ID,
			Name:       rest.Name,
			Address:    rest.Address,
			DistanceKm: math.Round(rest.DistanceKm*10) / 10,
		})
	}
	writeJSON(w, http.StatusOK, res)
}

// parseLatLng reads "12.9352,77.6245".
func parseLatLng(s string) (lat, lng float64, ok bool) {
	a, b, found := strings.Cut(s, ",")
	if !found {
		return 0, 0, false
	}
	lat, errLat := strconv.ParseFloat(a, 64)
	lng, errLng := strconv.ParseFloat(b, 64)
	return lat, lng, errLat == nil && errLng == nil
}

type dishJSON struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Price   int    `json:"price"`
	SoldOut bool   `json:"sold_out"`
}

// GET /v1/restaurants/{id}/menu returns the restaurant with its dishes inside it.
func (h *handler) menu(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	rest, dishes, err := h.svc.Restaurants.Menu(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	res := struct {
		ID     int64      `json:"id"`
		Name   string     `json:"name"`
		Dishes []dishJSON `json:"dishes"`
	}{ID: rest.ID, Name: rest.Name, Dishes: []dishJSON{}}
	for _, d := range dishes {
		res.Dishes = append(res.Dishes, dishJSON{ID: d.ID, Name: d.Name, Price: d.Price, SoldOut: d.SoldOut})
	}
	writeJSON(w, http.StatusOK, res)
}
