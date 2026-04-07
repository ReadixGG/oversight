package main

import (
	"flag"
	"log"
	"net/http"
	"oversight-server/internal/game"
	"oversight-server/internal/network"
)

func main() {
	addr := flag.String("addr", ":8080", "server listen address")
	tickRate := flag.Int("tick", 20, "server tick rate (Hz)")
	flag.Parse()

	hub := network.NewHub()
	go hub.Run()

	matchmaker := game.NewMatchmaker(hub, *tickRate)
	go matchmaker.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		network.ServeWS(hub, w, r)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	log.Printf("OverSight server starting on %s (tick rate: %d Hz)", *addr, *tickRate)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
