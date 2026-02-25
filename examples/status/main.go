package main

import (
	"fmt"
	"log"
	"os"

	go_monobank "github.com/stremovskyy/go-monobank"
)

func main() {
	token := os.Getenv("MONO_TOKEN")
	if token == "" {
		log.Fatal("set MONO_TOKEN env")
	}
	invoiceID := os.Getenv("INVOICE_ID")
	if invoiceID == "" {
		log.Fatal("set INVOICE_ID env")
	}

	client := go_monobank.NewDefaultClient()

	req := go_monobank.NewRequest().
		WithToken(token).
		WithInvoiceID(invoiceID)

	resp, err := client.Status(req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("invoiceId:", resp.InvoiceID)
	fmt.Println("status:", resp.Status)
	if resp.FailureReason != nil {
		fmt.Println("failureReason:", *resp.FailureReason)
	}
	if resp.ErrCode != nil {
		fmt.Println("errCode:", *resp.ErrCode)
	}
	if resp.WalletData != nil {
		fmt.Println("walletId:", resp.WalletData.WalletID)
		fmt.Println("cardToken:", resp.WalletData.CardToken)
		fmt.Println("walletStatus:", resp.WalletData.Status)
	}
}
