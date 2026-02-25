package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	go_monobank "github.com/stremovskyy/go-monobank"
	log2 "github.com/stremovskyy/go-monobank/log"
)

func main() {
	token := os.Getenv("MONO_TOKEN")
	if token == "" {
		log.Fatal("set MONO_TOKEN env")
	}

	client := go_monobank.NewClient(
		go_monobank.WithToken(token),
	)

	client.SetLogLevel(log2.Debug)

	h := func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad body"))
			return
		}

		xSign := r.Header.Get("X-Sign")
		event, err := client.ParseAndVerifyWebhook(body, xSign)
		if err != nil {
			// invalid signature or invalid JSON
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		fmt.Printf("webhook: invoiceId=%s status=%s\n", event.InvoiceID, event.Status)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}

	http.HandleFunc("/webhook", h)
	addr := ":8081"
	fmt.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
