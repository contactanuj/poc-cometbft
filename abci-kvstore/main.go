package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	abciserver "github.com/cometbft/cometbft/abci/server"
	cmtlog "github.com/cometbft/cometbft/libs/log"
)

func main() {
	addr := flag.String("addr", "tcp://0.0.0.0:36658", "ABCI listen address")
	dbPath := flag.String("db", "/data/kvstore-db", "Badger DB path")
	flag.Parse()

	log.Printf("[kvstore] main: starting up addr=%q dbPath=%q", *addr, *dbPath)

	state, err := NewState(*dbPath)
	if err != nil {
		log.Fatalf("[kvstore] main: failed to create state dbPath=%q err=%v", *dbPath, err)
	}
	defer state.Close()

	app := NewKVStoreApp(state)
	logger := cmtlog.NewTMLogger(cmtlog.NewSyncWriter(os.Stdout))

	server := abciserver.NewSocketServer(*addr, app)
	server.SetLogger(logger)

	if err := server.Start(); err != nil {
		log.Fatalf("[kvstore] main: failed to start ABCI server addr=%q err=%v", *addr, err)
	}
	log.Printf("[kvstore] main: ABCI kvstore server listening addr=%q", *addr)
	fmt.Printf("ABCI kvstore server listening on %s\n", *addr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	log.Printf("[kvstore] main: received signal=%v, shutting down", sig)
	if err := server.Stop(); err != nil {
		log.Printf("[kvstore] main: error stopping server err=%v", err)
	}
	log.Printf("[kvstore] main: shutdown complete")
}
