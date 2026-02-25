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

	resp, err := client.FiscalChecks(
		go_monobank.NewRequest().
			WithToken(token).
			WithInvoiceID(invoiceID),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("checks:", len(resp.Checks))
	fmt.Println("done:", len(resp.DoneChecks()))
	fmt.Println("pending:", len(resp.PendingChecks()))
	fmt.Println("failed:", len(resp.FailedChecks()))

	if last, ok := resp.LastCheck(); ok {
		fmt.Println("last_id:", last.ID)
		fmt.Println("last_type:", last.Type)
		fmt.Println("last_status:", last.Status)
		fmt.Println("last_status_description:", last.StatusDescription)
		fmt.Println("last_fiscalization_source:", last.FiscalizationSource)

		if taxURL, err := last.ParsedTaxURL(); err != nil {
			fmt.Println("tax_url_parse_error:", err)
		} else if taxURL != nil {
			fmt.Println("last_tax_url:", taxURL.String())
		}

		if fileBytes, err := last.DecodedFile(); err != nil {
			fmt.Println("file_decode_error:", err)
		} else {
			fmt.Println("last_file_bytes:", len(fileBytes))
		}
	}
}
