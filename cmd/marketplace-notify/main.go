package main

import (
	"fmt"
	"log"
	"net/http"

	marketplace "marketplace-notify"
)

func main() {
	client, err := marketplace.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	channel := "marketplace-orders"
	if err := client.CreateChannel(channel, "private", "infrai"); err != nil {
		log.Fatal(err)
	}
	handoff := marketplace.OrderHandoff{OrderID: "ord-demo", BuyerID: "buyer-demo", SellerID: "seller-demo", AssetID: "asset-demo"}
	if err := client.Publish(channel, "order.handoff", handoff.Notification(), handoff.SellerID); err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprintln(w, "ok") })
	log.Println("marketplace notification service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
