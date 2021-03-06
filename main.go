package main

import (
	"./build/hello"
	"fmt"

	"context"

	"math/big"
	"strings"
	"time"

	"github.com/chislab/go-fiscobcos/accounts/abi"
	"github.com/chislab/go-fiscobcos/accounts/abi/bind"
	"github.com/chislab/go-fiscobcos/common"
	"github.com/chislab/go-fiscobcos/core/types"
	"github.com/chislab/go-fiscobcos/crypto"
	"github.com/chislab/go-fiscobcos/ethclient"
)

var (
	defaultNode   = "http://192.168.31.220:8545"
	genesisKey, _ = crypto.HexToECDSA("526ccb243b5e279a3ce30c08e4d091a0eb2c3bb5a700946d4da47b28df8fe6d5")
)

func main() {
	gethCli, err := ethclient.Dial(defaultNode)
	if err != nil {
		return
	}
	genesisAuth := bind.NewKeyedTransactor(genesisKey)
	genesisAuth.GasLimit = 4700000
	height, err := gethCli.BlockNumber(context.Background(), 1)
	fmt.Println("BlockNumber: ", height.String())
	addr := rawDeploy(gethCli)
	testGetAndSet(gethCli, genesisAuth, *addr)
	testGetAndSet(gethCli, genesisAuth, *addr)
	testGetAndSet(gethCli, genesisAuth, *addr)
}

func rawDeploy(gethCli *ethclient.Client) *common.Address {
	contractABI, _ := abi.JSON(strings.NewReader(hello.HelloABI))
	contractBin := common.FromHex(hello.HelloBin)
	input, _ := contractABI.Pack("")
	payLoad := append(contractBin, input...)
	nonce := time.Now().Unix()
	//nonce := 10001
	height, err := gethCli.BlockNumber(context.Background(), 1)
	rawTx := types.NewContractCreation(uint64(nonce), height.Uint64()+100, big.NewInt(0),
		30000000, big.NewInt(30000000), payLoad, big.NewInt(1), big.NewInt(1), nil)

	genesisAuth := bind.NewKeyedTransactor(genesisKey)
	genesisAuth.GasLimit = 30000000
	signedTx, err := genesisAuth.Signer(types.HomesteadSigner{}, genesisAuth.From, rawTx)
	fmt.Println("txHash: ", signedTx.Hash().String())
	err = gethCli.SendTransaction(context.Background(), signedTx)
	if err != nil {
		fmt.Println(err.Error())
	}

	receipt, err := CheckTxStatus(signedTx.Hash().String(), gethCli)
	contractAddr := receipt.ContractAddress
	return &contractAddr
}

func testGetAndSet(gethCli *ethclient.Client, opts *bind.TransactOpts, address common.Address) {
	deployedHello, err := hello.NewHello(address, gethCli)
	if err != nil {
		return
	}
	fmt.Println("test Get() method")
	r, err := deployedHello.Get(&bind.CallOpts{GroupId: opts.GroupId, From: opts.From})
	fmt.Println(r)
	opts.RandomId = big.NewInt(time.Now().Unix())
	height, _ := gethCli.BlockNumber(context.Background(), 1)
	opts.BlockLimit = big.NewInt(0).Add(height, big.NewInt(100))
	fmt.Println("test Set() method")
	tx, err := deployedHello.Set(opts, time.Now().String())
	if err != nil {
		return
	}
	fmt.Println("Set() txHash: ", tx.Hash().String())
	time.Sleep(1 * time.Second)
	fmt.Println("test Get() method again")
	r, err = deployedHello.Get(&bind.CallOpts{GroupId: opts.GroupId, From: opts.From})
	fmt.Println(r)
}

func CheckTxStatus(txHash string, gethCli *ethclient.Client) (*types.Receipt, error) {
	receipt, err := WaitMinedByHash(context.Background(), gethCli, txHash)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

func WaitMinedByHash(ctx context.Context, gethCli *ethclient.Client, txHash string) (*types.Receipt, error) {
	queryTicker := time.NewTicker(time.Second)
	defer queryTicker.Stop()
	for {
		tx, _ := gethCli.TransactionReceipt(ctx, 1, common.HexToHash(txHash))
		if tx != nil {
			return tx, nil
		}
		// Wait for the next round.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-queryTicker.C:
		}
	}
}

func UnlockPocket(keyJson, keyPwd string) (auth *bind.TransactOpts, err error) {
	myAuth, err := bind.NewTransactor(strings.NewReader(keyJson), keyPwd)
	if err != nil {
		return nil, err
	}
	if myAuth.GasPrice == nil {
		myAuth.GasPrice = big.NewInt(0)
	}
	return myAuth, nil
}
