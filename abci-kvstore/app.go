package main

import (
	"bytes"
	"context"
	"fmt"
	"log"

	abcitypes "github.com/cometbft/cometbft/abci/types"
)

type KVStoreApp struct {
	abcitypes.BaseApplication
	state   *State
	height  int64
	appHash []byte
	staged  [][2][]byte
}

func NewKVStoreApp(state *State) *KVStoreApp {
	height, appHash, err := state.LoadMeta()
	if err != nil {
		log.Fatalf("[kvstore] NewKVStoreApp: failed to load meta: %v", err)
	}
	log.Printf("[kvstore] NewKVStoreApp: restored state height=%d appHash=%x", height, appHash)
	return &KVStoreApp{
		state:   state,
		height:  height,
		appHash: appHash,
	}
}

func (app *KVStoreApp) Info(_ context.Context, req *abcitypes.RequestInfo) (*abcitypes.ResponseInfo, error) {
	log.Printf("[kvstore] Info: height=%d appHash=%x", app.height, app.appHash)
	return &abcitypes.ResponseInfo{
		Data:             "kvstore",
		Version:          "1.0.0",
		AppVersion:       1,
		LastBlockHeight:  app.height,
		LastBlockAppHash: app.appHash,
	}, nil
}

func (app *KVStoreApp) Query(_ context.Context, req *abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	val, err := app.state.Get(req.Data)
	if err != nil {
		log.Printf("[kvstore] Query: key=%q found=false err=%v", string(req.Data), err)
		return &abcitypes.ResponseQuery{
			Code: 1,
			Log:  fmt.Sprintf("key not found: %s", string(req.Data)),
		}, nil
	}
	log.Printf("[kvstore] Query: key=%q found=true valueLen=%d", string(req.Data), len(val))
	return &abcitypes.ResponseQuery{
		Code:  0,
		Key:   req.Data,
		Value: val,
	}, nil
}

func (app *KVStoreApp) CheckTx(_ context.Context, req *abcitypes.RequestCheckTx) (*abcitypes.ResponseCheckTx, error) {
	valid := isValidTx(req.Tx)
	log.Printf("[kvstore] CheckTx: tx=%q valid=%t", string(req.Tx), valid)
	if !valid {
		return &abcitypes.ResponseCheckTx{Code: 1, Log: "invalid tx format, expected key=value"}, nil
	}
	return &abcitypes.ResponseCheckTx{Code: 0}, nil
}

func (app *KVStoreApp) FinalizeBlock(_ context.Context, req *abcitypes.RequestFinalizeBlock) (*abcitypes.ResponseFinalizeBlock, error) {
	log.Printf("[kvstore] FinalizeBlock: height=%d txCount=%d", req.Height, len(req.Txs))
	app.staged = nil
	txResults := make([]*abcitypes.ExecTxResult, len(req.Txs))
	invalidCount := 0
	for i, tx := range req.Txs {
		if !isValidTx(tx) {
			log.Printf("[kvstore] FinalizeBlock: tx[%d]=%q invalid", i, string(tx))
			txResults[i] = &abcitypes.ExecTxResult{Code: 1, Log: "invalid tx"}
			invalidCount++
			continue
		}
		parts := bytes.SplitN(tx, []byte("="), 2)
		log.Printf("[kvstore] FinalizeBlock: tx[%d] key=%q valueLen=%d", i, string(parts[0]), len(parts[1]))
		app.staged = append(app.staged, [2][]byte{parts[0], parts[1]})
		txResults[i] = &abcitypes.ExecTxResult{Code: 0}
	}
	app.height = req.Height
	log.Printf("[kvstore] FinalizeBlock: completed height=%d staged=%d invalid=%d", app.height, len(app.staged), invalidCount)
	return &abcitypes.ResponseFinalizeBlock{TxResults: txResults}, nil
}

func (app *KVStoreApp) Commit(_ context.Context, req *abcitypes.RequestCommit) (*abcitypes.ResponseCommit, error) {
	log.Printf("[kvstore] Commit: height=%d staged=%d", app.height, len(app.staged))
	if len(app.staged) > 0 {
		if err := app.state.BatchSet(app.staged); err != nil {
			log.Printf("[kvstore] Commit: batch set error height=%d err=%v", app.height, err)
		}
	}
	app.appHash = app.state.Hash()
	if err := app.state.SaveMeta(app.height, app.appHash); err != nil {
		log.Printf("[kvstore] Commit: save meta error height=%d err=%v", app.height, err)
	}
	log.Printf("[kvstore] Commit: completed height=%d newAppHash=%x", app.height, app.appHash)
	app.staged = nil
	return &abcitypes.ResponseCommit{}, nil
}

func (app *KVStoreApp) PrepareProposal(_ context.Context, req *abcitypes.RequestPrepareProposal) (*abcitypes.ResponsePrepareProposal, error) {
	log.Printf("[kvstore] PrepareProposal: height=%d txCount=%d maxTxBytes=%d", req.Height, len(req.Txs), req.MaxTxBytes)
	return &abcitypes.ResponsePrepareProposal{Txs: req.Txs}, nil
}

func (app *KVStoreApp) ProcessProposal(_ context.Context, req *abcitypes.RequestProcessProposal) (*abcitypes.ResponseProcessProposal, error) {
	log.Printf("[kvstore] ProcessProposal: height=%d txCount=%d status=ACCEPT", req.Height, len(req.Txs))
	return &abcitypes.ResponseProcessProposal{Status: abcitypes.ResponseProcessProposal_ACCEPT}, nil
}

func isValidTx(tx []byte) bool {
	parts := bytes.SplitN(tx, []byte("="), 2)
	return len(parts) == 2 && len(parts[0]) > 0
}
