package handler_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRestaurantsNearListsOnlyThoseInRangeNearestFirst(t *testing.T) {
	a := newApp(t)
	indiranagar := a.restaurant("Udupi Tiffin Room", 12.9719, 77.6412) // about 4.4 km from the customer
	koramangala := a.restaurant("Biryani House", 12.9352, 77.6245)     // next door
	a.restaurant("Mithai Mahal", 13.1007, 77.5963)                     // Yelahanka, about 19 km

	res := a.do("GET", "/v1/restaurants?near="+url.QueryEscape("12.9350,77.6240"), "", nil)

	assert.Equal(t, http.StatusOK, res.status)
	var out struct {
		Restaurants []struct {
			ID         int64   `json:"id"`
			DistanceKm float64 `json:"distance_km"`
		} `json:"restaurants"`
	}
	res.decode(&out)
	if assert.Len(t, out.Restaurants, 2) {
		assert.Equal(t, koramangala, out.Restaurants[0].ID)
		assert.Equal(t, indiranagar, out.Restaurants[1].ID)
		assert.InDelta(t, 4.4, out.Restaurants[1].DistanceKm, 0.2)
	}
}

func TestRestaurantsNearComesInPagesOf20(t *testing.T) {
	a := newApp(t)
	var nearestFirst []int64
	for i := range 25 { // each about 110 m further north than the one before
		nearestFirst = append(nearestFirst, a.restaurant(fmt.Sprintf("Restaurant %d", i), 12.9352+float64(i)*0.001, 77.6245))
	}

	page := func(query string) []int64 {
		res := a.do("GET", "/v1/restaurants?near="+url.QueryEscape("12.9352,77.6245")+query, "", nil)
		assert.Equal(t, http.StatusOK, res.status)
		var out struct {
			Restaurants []struct {
				ID int64 `json:"id"`
			} `json:"restaurants"`
		}
		res.decode(&out)
		var ids []int64
		for _, r := range out.Restaurants {
			ids = append(ids, r.ID)
		}
		return ids
	}

	assert.Equal(t, nearestFirst[:20], page(""))
	assert.Equal(t, nearestFirst[:20], page("&page=1"))
	assert.Equal(t, nearestFirst[20:], page("&page=2"))
	assert.Empty(t, page("&page=3"))
}

func TestRestaurantsNearNeedsAValidSpotAndPage(t *testing.T) {
	a := newApp(t)

	for _, near := range []string{"", "koramangala", "12.93", "95,77.62"} {
		res := a.do("GET", "/v1/restaurants?near="+url.QueryEscape(near), "", nil)
		assert.Equal(t, http.StatusBadRequest, res.status, "near=%q", near)
	}
	for _, page := range []string{"0", "-1", "two"} {
		res := a.do("GET", "/v1/restaurants?near=12.9352,77.6245&page="+page, "", nil)
		assert.Equal(t, http.StatusBadRequest, res.status, "page=%q", page)
	}
}

func TestMenuHasOnlyThatRestaurantsDishesWithSoldOutOnesMarked(t *testing.T) {
	a := newApp(t)
	biryaniHouse := a.restaurant("Biryani House", 12.9352, 77.6245)
	udupi := a.restaurant("Udupi Tiffin Room", 12.9719, 77.6412)
	biryani := a.dish(biryaniHouse, "Chicken biryani", 25000)
	raita := a.dish(biryaniHouse, "Raita", 4000)
	a.dish(udupi, "Masala dosa", 9000)
	a.soldOut(raita)

	res := a.do("GET", fmt.Sprintf("/v1/restaurants/%d/menu", biryaniHouse), "", nil)

	assert.Equal(t, http.StatusOK, res.status)
	var menu struct {
		ID     int64 `json:"id"`
		Dishes []struct {
			ID      int64 `json:"id"`
			Price   int   `json:"price"`
			SoldOut bool  `json:"sold_out"`
		} `json:"dishes"`
	}
	res.decode(&menu)
	assert.Equal(t, biryaniHouse, menu.ID)
	if assert.Len(t, menu.Dishes, 2) {
		assert.Equal(t, biryani, menu.Dishes[0].ID)
		assert.Equal(t, 25000, menu.Dishes[0].Price)
		assert.False(t, menu.Dishes[0].SoldOut)
		assert.Equal(t, raita, menu.Dishes[1].ID)
		assert.True(t, menu.Dishes[1].SoldOut)
	}

	res = a.do("GET", "/v1/restaurants/999999999/menu", "", nil)
	assert.Equal(t, http.StatusNotFound, res.status)
}
