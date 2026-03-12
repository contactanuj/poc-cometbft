package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	taskregtypes "poc-cometbft/mychain/x/taskreg/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ChainClient struct {
	grpcConn  *grpc.ClientConn
	rpcClient *rpchttp.HTTP
	cdc       codec.Codec
	ir        codectypes.InterfaceRegistry
	txConfig  sdkclient.TxConfig
	chainID   string
	kr        keyring.Keyring
	keyName   string
	mu        sync.Mutex

	taskregQueryClient taskregtypes.QueryClient
	wasmQueryClient    wasmtypes.QueryClient
	authQueryClient    authtypes.QueryClient

	contractAddr string
}

func NewChainClient(
	grpcAddr, rpcAddr, chainID, mnemonic string,
	cdc codec.Codec,
	ir codectypes.InterfaceRegistry,
	txCfg sdkclient.TxConfig,
) (*ChainClient, error) {
	// gRPC connection
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}

	// RPC client
	rpc, err := rpchttp.New(rpcAddr, "/websocket")
	if err != nil {
		return nil, fmt.Errorf("rpc client: %w", err)
	}

	// In-memory keyring with test key
	kr := keyring.NewInMemory(cdc)
	hdPath := hd.CreateHDPath(118, 0, 0).String()
	_, err = kr.NewAccount("admin", mnemonic, "", hdPath, hd.Secp256k1)
	if err != nil {
		return nil, fmt.Errorf("import key: %w", err)
	}

	c := &ChainClient{
		grpcConn:           conn,
		rpcClient:          rpc,
		cdc:                cdc,
		ir:                 ir,
		txConfig:           txCfg,
		chainID:            chainID,
		kr:                 kr,
		keyName:            "admin",
		taskregQueryClient: taskregtypes.NewQueryClient(conn),
		wasmQueryClient:    wasmtypes.NewQueryClient(conn),
		authQueryClient:    authtypes.NewQueryClient(conn),
	}

	return c, nil
}

func (c *ChainClient) SetContractAddr(addr string) {
	c.contractAddr = addr
}

func (c *ChainClient) CheckHealth(ctx context.Context) error {
	log.Printf("[chain-client] CheckHealth: querying node status")
	_, err := c.rpcClient.Status(ctx)
	if err != nil {
		log.Printf("[chain-client] CheckHealth: error=%v", err)
	} else {
		log.Printf("[chain-client] CheckHealth: ok")
	}
	return err
}

func (c *ChainClient) getSignerAddr() (sdk.AccAddress, error) {
	rec, err := c.kr.Key(c.keyName)
	if err != nil {
		return nil, err
	}
	addr, err := rec.GetAddress()
	if err != nil {
		return nil, err
	}
	return addr, nil
}

func (c *ChainClient) getAccount(ctx context.Context, addr sdk.AccAddress) (authtypes.AccountI, error) {
	log.Printf("[chain-client] getAccount: addr=%s", addr.String())
	resp, err := c.authQueryClient.Account(ctx, &authtypes.QueryAccountRequest{
		Address: addr.String(),
	})
	if err != nil {
		log.Printf("[chain-client] getAccount: error=%v", err)
		return nil, err
	}
	var acc authtypes.AccountI
	if err := c.ir.UnpackAny(resp.Account, &acc); err != nil {
		log.Printf("[chain-client] getAccount: unpack error=%v", err)
		return nil, err
	}
	log.Printf("[chain-client] getAccount: seq=%d accNum=%d", acc.GetSequence(), acc.GetAccountNumber())
	return acc, nil
}

func (c *ChainClient) broadcastMsg(ctx context.Context, msgs ...sdk.Msg) (*sdk.TxResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	addr, err := c.getSignerAddr()
	if err != nil {
		return nil, fmt.Errorf("get signer: %w", err)
	}

	log.Printf("[chain-client] BroadcastMsg: msgCount=%d signer=%s", len(msgs), addr.String())

	acc, err := c.getAccount(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	// Use TxFactory for signing
	factory := clienttx.Factory{}.
		WithChainID(c.chainID).
		WithKeybase(c.kr).
		WithTxConfig(c.txConfig).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithAccountNumber(acc.GetAccountNumber()).
		WithSequence(acc.GetSequence()).
		WithGas(500000).
		WithFees("0stake")

	txBuilder, err := factory.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, fmt.Errorf("build tx: %w", err)
	}

	err = clienttx.Sign(ctx, factory, c.keyName, txBuilder, true)
	if err != nil {
		log.Printf("[chain-client] BroadcastMsg: sign error=%v", err)
		return nil, fmt.Errorf("sign: %w", err)
	}
	log.Printf("[chain-client] BroadcastMsg: tx signed successfully")

	txBytes, err := c.txConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("encode tx: %w", err)
	}

	result, err := c.rpcClient.BroadcastTxSync(ctx, txBytes)
	if err != nil {
		log.Printf("[chain-client] BroadcastMsg: broadcast error=%v", err)
		return nil, fmt.Errorf("broadcast: %w", err)
	}

	if result.Code != 0 {
		log.Printf("[chain-client] BroadcastMsg: tx failed code=%d log=%s", result.Code, result.Log)
		return nil, fmt.Errorf("tx failed (code %d): %s", result.Code, result.Log)
	}

	log.Printf("[chain-client] BroadcastMsg: success txHash=%s", result.Hash.String())
	return &sdk.TxResponse{
		TxHash: result.Hash.String(),
		Code:   result.Code,
	}, nil
}

// === SDK Module: Task operations ===

func (c *ChainClient) CreateTask(ctx context.Context, title, description string) (map[string]interface{}, error) {
	log.Printf("[chain-client] CreateTask: title=%q", title)
	addr, err := c.getSignerAddr()
	if err != nil {
		return nil, err
	}
	msg := &taskregtypes.MsgCreateTask{
		Creator:     addr.String(),
		Title:       title,
		Description: description,
	}
	resp, err := c.broadcastMsg(ctx, msg)
	if err != nil {
		log.Printf("[chain-client] CreateTask: error=%v", err)
		return nil, err
	}
	log.Printf("[chain-client] CreateTask: success txHash=%s", resp.TxHash)
	return map[string]interface{}{
		"tx_hash": resp.TxHash,
	}, nil
}

func (c *ChainClient) GetTask(ctx context.Context, id uint64) (interface{}, error) {
	log.Printf("[chain-client] GetTask: id=%d", id)
	resp, err := c.taskregQueryClient.Task(ctx, &taskregtypes.QueryTaskRequest{Id: id})
	if err != nil {
		log.Printf("[chain-client] GetTask: error=%v", err)
		return nil, err
	}
	log.Printf("[chain-client] GetTask: found id=%d", id)
	return resp.Task, nil
}

func (c *ChainClient) ListTasks(ctx context.Context) (interface{}, error) {
	log.Printf("[chain-client] ListTasks: querying all tasks")
	resp, err := c.taskregQueryClient.ListTasks(ctx, &taskregtypes.QueryListTasksRequest{})
	if err != nil {
		log.Printf("[chain-client] ListTasks: error=%v", err)
		return nil, err
	}
	log.Printf("[chain-client] ListTasks: found %d tasks", len(resp.Tasks))
	return resp.Tasks, nil
}

func (c *ChainClient) AssignTask(ctx context.Context, taskID uint64, assignee string) (map[string]interface{}, error) {
	log.Printf("[chain-client] AssignTask: taskID=%d assignee=%s", taskID, assignee)
	addr, err := c.getSignerAddr()
	if err != nil {
		return nil, err
	}
	msg := &taskregtypes.MsgAssignTask{
		Creator:  addr.String(),
		TaskId:   taskID,
		Assignee: assignee,
	}
	resp, err := c.broadcastMsg(ctx, msg)
	if err != nil {
		log.Printf("[chain-client] AssignTask: error=%v", err)
		return nil, err
	}
	log.Printf("[chain-client] AssignTask: success txHash=%s", resp.TxHash)
	return map[string]interface{}{
		"tx_hash": resp.TxHash,
	}, nil
}

func (c *ChainClient) CompleteTask(ctx context.Context, taskID uint64) (map[string]interface{}, error) {
	log.Printf("[chain-client] CompleteTask: taskID=%d", taskID)
	addr, err := c.getSignerAddr()
	if err != nil {
		return nil, err
	}
	msg := &taskregtypes.MsgCompleteTask{
		Assignee: addr.String(),
		TaskId:   taskID,
	}
	resp, err := c.broadcastMsg(ctx, msg)
	if err != nil {
		log.Printf("[chain-client] CompleteTask: error=%v", err)
		return nil, err
	}
	log.Printf("[chain-client] CompleteTask: success txHash=%s", resp.TxHash)
	return map[string]interface{}{
		"tx_hash": resp.TxHash,
	}, nil
}

// === CosmWasm: Bounty operations ===

func (c *ChainClient) FundBounty(ctx context.Context, taskID uint64, amount, denom string) (map[string]interface{}, error) {
	log.Printf("[chain-client] FundBounty: taskID=%d amount=%s denom=%s", taskID, amount, denom)
	addr, err := c.getSignerAddr()
	if err != nil {
		return nil, err
	}

	executeMsg, _ := json.Marshal(map[string]interface{}{
		"fund_bounty": map[string]interface{}{
			"task_id": taskID,
		},
	})

	coin, err := sdk.ParseCoinNormalized(amount + denom)
	if err != nil {
		log.Printf("[chain-client] FundBounty: parse coin error=%v", err)
		return nil, fmt.Errorf("parse coin: %w", err)
	}

	msg := &wasmtypes.MsgExecuteContract{
		Sender:   addr.String(),
		Contract: c.contractAddr,
		Msg:      executeMsg,
		Funds:    sdk.NewCoins(coin),
	}

	resp, err := c.broadcastMsg(ctx, msg)
	if err != nil {
		log.Printf("[chain-client] FundBounty: error=%v", err)
		return nil, err
	}
	log.Printf("[chain-client] FundBounty: success txHash=%s", resp.TxHash)
	return map[string]interface{}{
		"tx_hash": resp.TxHash,
	}, nil
}

func (c *ChainClient) ClaimBounty(ctx context.Context, taskID uint64) (map[string]interface{}, error) {
	log.Printf("[chain-client] ClaimBounty: taskID=%d", taskID)
	addr, err := c.getSignerAddr()
	if err != nil {
		return nil, err
	}

	executeMsg, _ := json.Marshal(map[string]interface{}{
		"claim_bounty": map[string]interface{}{
			"task_id": taskID,
		},
	})

	msg := &wasmtypes.MsgExecuteContract{
		Sender:   addr.String(),
		Contract: c.contractAddr,
		Msg:      executeMsg,
	}

	resp, err := c.broadcastMsg(ctx, msg)
	if err != nil {
		log.Printf("[chain-client] ClaimBounty: error=%v", err)
		return nil, err
	}
	log.Printf("[chain-client] ClaimBounty: success txHash=%s", resp.TxHash)
	return map[string]interface{}{
		"tx_hash": resp.TxHash,
	}, nil
}

func (c *ChainClient) GetBounty(ctx context.Context, taskID uint64) (interface{}, error) {
	log.Printf("[chain-client] GetBounty: taskID=%d", taskID)
	queryMsg, _ := json.Marshal(map[string]interface{}{
		"bounty": map[string]interface{}{
			"task_id": taskID,
		},
	})
	result, err := c.queryContract(ctx, queryMsg)
	if err != nil {
		log.Printf("[chain-client] GetBounty: error=%v", err)
	} else {
		log.Printf("[chain-client] GetBounty: found taskID=%d", taskID)
	}
	return result, err
}

func (c *ChainClient) ListBounties(ctx context.Context) (interface{}, error) {
	log.Printf("[chain-client] ListBounties: querying all bounties")
	queryMsg, _ := json.Marshal(map[string]interface{}{
		"list_bounties": map[string]interface{}{},
	})
	result, err := c.queryContract(ctx, queryMsg)
	if err != nil {
		log.Printf("[chain-client] ListBounties: error=%v", err)
	} else {
		log.Printf("[chain-client] ListBounties: success")
	}
	return result, err
}

func (c *ChainClient) GetContractConfig(ctx context.Context) (interface{}, error) {
	log.Printf("[chain-client] GetContractConfig: querying config")
	queryMsg, _ := json.Marshal(map[string]interface{}{
		"config": map[string]interface{}{},
	})
	result, err := c.queryContract(ctx, queryMsg)
	if err != nil {
		log.Printf("[chain-client] GetContractConfig: error=%v", err)
	} else {
		log.Printf("[chain-client] GetContractConfig: success")
	}
	return result, err
}

func (c *ChainClient) queryContract(ctx context.Context, queryMsg []byte) (interface{}, error) {
	log.Printf("[chain-client] queryContract: contract=%s query=%s", c.contractAddr, string(queryMsg))
	resp, err := c.wasmQueryClient.SmartContractState(ctx, &wasmtypes.QuerySmartContractStateRequest{
		Address:   c.contractAddr,
		QueryData: queryMsg,
	})
	if err != nil {
		log.Printf("[chain-client] queryContract: error=%v", err)
		return nil, err
	}
	var result interface{}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		log.Printf("[chain-client] queryContract: unmarshal error=%v", err)
		return nil, err
	}
	log.Printf("[chain-client] queryContract: success")
	return result, nil
}

func (c *ChainClient) ResolveContractByCodeID(ctx context.Context, codeID uint64) (string, error) {
	log.Printf("[chain-client] ResolveContract: codeID=%d", codeID)
	resp, err := c.wasmQueryClient.ContractsByCode(ctx, &wasmtypes.QueryContractsByCodeRequest{
		CodeId: codeID,
	})
	if err != nil {
		log.Printf("[chain-client] ResolveContract: error=%v", err)
		return "", fmt.Errorf("query contracts by code: %w", err)
	}
	if len(resp.Contracts) == 0 {
		log.Printf("[chain-client] ResolveContract: no contracts found for codeID=%d", codeID)
		return "", fmt.Errorf("no contracts found for code ID %d", codeID)
	}
	log.Printf("[chain-client] ResolveContract: codeID=%d addr=%s", codeID, resp.Contracts[0])
	return resp.Contracts[0], nil
}
