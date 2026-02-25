package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	go_monobank "github.com/stremovskyy/go-monobank"
)

func main() {
	token := os.Getenv("MONO_TOKEN")
	if token == "" {
		log.Fatal("set MONO_TOKEN env")
	}
	cardToken := os.Getenv("CARD_TOKEN")
	if cardToken == "" {
		log.Fatal("set CARD_TOKEN env")
	}

	amount := int64(4200)
	if s := os.Getenv("AMOUNT"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			log.Fatalf("invalid AMOUNT %q: %v", s, err)
		}
		amount = v
	}

	client := go_monobank.NewDefaultClient()

	req := go_monobank.NewRequest().
		WithToken(token).
		WithCardToken(cardToken).
		WithAmount(amount).
		WithCurrency(go_monobank.CurrencyUAH).
		WithInitiationKind(go_monobank.InitiationMerchant).
		WithReference("wallet-pay-001").
		WithDestination("Token payment")

	resp, err := client.Payment(req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("invoiceId:", resp.InvoiceID)
	fmt.Println("status:", resp.Status)
	if resp.TDSURL != nil {
		fmt.Println("tdsUrl:", *resp.TDSURL)
	}
	if resp.FailureReason != nil {
		fmt.Println("failureReason:", *resp.FailureReason)
	}
}
