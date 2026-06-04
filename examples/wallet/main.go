package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	go_monobank "github.com/stremovskyy/go-monobank"
)

func main() {
	token := os.Getenv("MONO_TOKEN")
	if token == "" {
		log.Fatal("set MONO_TOKEN env")
	}

	walletID := os.Getenv("WALLET_ID")
	if walletID == "" {
		log.Fatal("set WALLET_ID env")
	}

	cardToken := strings.TrimSpace(os.Getenv("CARD_TOKEN"))

	client := go_monobank.NewDefaultClient()

	req := go_monobank.NewRequest().
		WithToken(token).
		WithWalletID(walletID)

	resp, err := client.Wallet(req)
	if err != nil {
		log.Fatal(err)
	}

	if resp == nil || len(resp.Wallet) == 0 {
		fmt.Println("wallet is empty")
		return
	}

	found := false
	for _, card := range resp.Wallet {
		if cardToken != "" && card.CardToken != cardToken {
			continue
		}

		found = true
		printCard(card)
	}

	if cardToken != "" && !found {
		log.Fatalf("CARD_TOKEN %q was not found in WALLET_ID %q", cardToken, walletID)
	}
}

func printCard(card go_monobank.WalletItem) {
	fmt.Println("cardToken:", card.CardToken)
	fmt.Println("maskedPan:", card.MaskedPan)
	fmt.Println("displayMaskedPan:", displayMaskedPan(card.MaskedPan))
	fmt.Println("country:", card.Country)
}

func displayMaskedPan(maskedPan string) string {
	return strings.ReplaceAll(strings.TrimSpace(maskedPan), "*", "X")
}
