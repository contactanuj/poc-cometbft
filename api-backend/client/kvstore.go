package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
)

type KVStoreClient struct {
	rpcClient *rpchttp.HTTP
}

func NewKVStoreClient(rpcAddr string) (*KVStoreClient, error) {
	client, err := rpchttp.New(rpcAddr, "/websocket")
	if err != nil {
		return nil, fmt.Errorf("failed to create KV RPC client: %w", err)
	}
	return &KVStoreClient{rpcClient: client}, nil
}

func (c *KVStoreClient) Set(ctx context.Context, key, value string) (map[string]string, error) {
	log.Printf("[kv-client] Set: key=%q valueLen=%d", key, len(value))
	tx := []byte(fmt.Sprintf("%s=%s", key, value))
	result, err := c.rpcClient.BroadcastTxCommit(ctx, tx)
	if err != nil {
		log.Printf("[kv-client] Set: broadcast error=%v", err)
		return nil, fmt.Errorf("broadcast tx: %w", err)
	}
	if result.TxResult.Code != 0 {
		log.Printf("[kv-client] Set: tx failed code=%d log=%s", result.TxResult.Code, result.TxResult.Log)
		return nil, fmt.Errorf("tx failed: %s", result.TxResult.Log)
	}
	txHash := hex.EncodeToString(result.Hash)
	log.Printf("[kv-client] Set: success key=%q txHash=%s", key, txHash)
	return map[string]string{
		"tx_hash": txHash,
		"key":     key,
		"value":   value,
	}, nil
}

func (c *KVStoreClient) Get(ctx context.Context, key string) (string, error) {
	log.Printf("[kv-client] Get: key=%q", key)
	result, err := c.rpcClient.ABCIQuery(ctx, "", []byte(key))
	if err != nil {
		log.Printf("[kv-client] Get: ABCI query error=%v", err)
		return "", fmt.Errorf("ABCI query: %w", err)
	}
	if result.Response.Code != 0 {
		log.Printf("[kv-client] Get: key=%q not found log=%s", key, result.Response.Log)
		return "", fmt.Errorf("key not found: %s", result.Response.Log)
	}
	found := len(result.Response.Value) > 0
	log.Printf("[kv-client] Get: key=%q found=%t valueLen=%d", key, found, len(result.Response.Value))
	return string(result.Response.Value), nil
}

func (c *KVStoreClient) CheckHealth(ctx context.Context) error {
	log.Printf("[kv-client] CheckHealth: querying node status")
	_, err := c.rpcClient.Status(ctx)
	if err != nil {
		log.Printf("[kv-client] CheckHealth: error=%v", err)
	} else {
		log.Printf("[kv-client] CheckHealth: ok")
	}
	return err
}

func (c *KVStoreClient) Status(ctx context.Context) (*coretypes.ResultStatus, error) {
	return c.rpcClient.Status(ctx)
}
