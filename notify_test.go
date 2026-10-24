package marketplace

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRetryRejected429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"ok":false,"error":{"code":"RATE_LIMITED"}}`)
			return
		}
		fmt.Fprint(w, `{"ok":true,"data":{"sent":true}}`)
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, Key: "test", HTTP: server.Client()}
	if err := client.Publish("orders", "handoff", nil, "seller"); err != nil {
		t.Fatalf("publish after 429: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected two attempts, got %d", attempts)
	}
}

func TestOrderHandoffNotification(t *testing.T) {
	got := (OrderHandoff{OrderID: "ord-7", BuyerID: "buyer-2", SellerID: "seller-9", AssetID: "asset-3"}).Notification()
	if got["state"] != "handed_to_seller" || got["order_id"] != "ord-7" {
		t.Fatalf("unexpected notification: %#v", got)
	}
}
