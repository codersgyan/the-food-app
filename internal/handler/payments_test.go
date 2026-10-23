package handler_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTappingPayManyTimesAtOnceChargesOnce(t *testing.T) {
	a := newApp(t)
	k := a.kitchen()
	id := a.placeOrder(k)

	// Ten taps on Pay, all arriving at the same moment.
	statuses := make([]int, 10)
	var wg sync.WaitGroup
	for i := range statuses {
		wg.Go(func() {
			statuses[i] = a.do("POST", fmt.Sprintf("/v1/orders/%d/pay", id), k.customer, nil).status
		})
	}
	wg.Wait()

	assert.Equal(t, 1, count(statuses, http.StatusOK))
	assert.Equal(t, 9, count(statuses, http.StatusConflict))
	assert.Equal(t, int32(1), a.payments.charges.Load())
}

func TestCustomerCannotPayForSomeoneElsesOrder(t *testing.T) {
	a := newApp(t)
	k := a.kitchen()
	id := a.placeOrder(k)
	vikram, _ := a.customer("+919800000002")

	res := a.do("POST", fmt.Sprintf("/v1/orders/%d/pay", id), vikram, nil)

	assert.Equal(t, http.StatusNotFound, res.status)
	assert.Equal(t, int32(0), a.payments.charges.Load())
}

func count(list []int, v int) int {
	n := 0
	for _, x := range list {
		if x == v {
			n++
		}
	}
	return n
}
