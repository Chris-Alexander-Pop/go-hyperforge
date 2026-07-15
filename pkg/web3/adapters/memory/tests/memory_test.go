package memory_test

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3/adapters/memory"
)

func TestClient_TransferAndBalance(t *testing.T) {
	ctx := context.Background()
	from := "0x1111111111111111111111111111111111111111"
	to := "0x2222222222222222222222222222222222222222"
	c := memory.NewClient(memory.ClientConfig{
		Address: from,
		InitialBalances: map[string]*big.Int{
			from: big.NewInt(1_000_000),
		},
	})
	defer c.Close()

	bal, err := c.GetBalance(ctx, from)
	if err != nil || bal.Cmp(big.NewInt(1_000_000)) != 0 {
		t.Fatalf("bal=%v err=%v", bal, err)
	}

	txHash, err := c.Transfer(ctx, to, big.NewInt(250_000))
	if err != nil {
		t.Fatal(err)
	}
	if txHash == "" {
		t.Fatal("expected tx hash")
	}

	fromBal, _ := c.GetBalance(ctx, from)
	toBal, _ := c.GetBalance(ctx, to)
	if fromBal.Cmp(big.NewInt(750_000)) != 0 || toBal.Cmp(big.NewInt(250_000)) != 0 {
		t.Fatalf("from=%s to=%s", fromBal, toBal)
	}

	receipt, err := c.GetTransactionReceipt(ctx, txHash)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != 1 || receipt.GasUsed != 21000 {
		t.Fatalf("receipt=%+v", receipt)
	}

	waited, err := c.WaitForTransaction(ctx, txHash)
	if err != nil || waited.TxHash != txHash {
		t.Fatalf("wait=%+v err=%v", waited, err)
	}
}

func TestClient_InsufficientFundsAndNoReceipt(t *testing.T) {
	ctx := context.Background()
	c := memory.NewClient(memory.ClientConfig{
		Address: "0xaaa",
		InitialBalances: map[string]*big.Int{
			"0xaaa": big.NewInt(10),
		},
	})
	_, err := c.Transfer(ctx, "0xbbb", big.NewInt(100))
	if err == nil {
		t.Fatal("expected insufficient funds")
	}
	if !pkgerrors.IsCode(err, web3.CodeRPCFailed) {
		t.Fatalf("code=%s", pkgerrors.Code(err))
	}

	_, err = c.GetTransactionReceipt(ctx, "0xmissing")
	if !pkgerrors.IsCode(err, web3.CodeNotFound) {
		t.Fatalf("code=%s", pkgerrors.Code(err))
	}
}

func TestClient_ChainBlockGasContract(t *testing.T) {
	ctx := context.Background()
	c := memory.NewClient(memory.ClientConfig{ChainID: 137, StartBlock: 50})
	id, err := c.GetChainID(ctx)
	if err != nil || id.Int64() != 137 {
		t.Fatalf("id=%v err=%v", id, err)
	}
	n, err := c.GetBlockNumber(ctx)
	if err != nil || n != 50 {
		t.Fatalf("block=%d err=%v", n, err)
	}
	gas, err := c.EstimateGas(ctx, "0x1", nil)
	if err != nil || gas != 21000 {
		t.Fatalf("gas=%d err=%v", gas, err)
	}
	gas, err = c.EstimateGas(ctx, "0x1", []byte{1, 2})
	if err != nil || gas != 100000 {
		t.Fatalf("gas=%d err=%v", gas, err)
	}

	c.SetContractResponse("0xContract", []byte{9, 8, 7})
	out, err := c.CallContract(ctx, "0xcontract", []byte{1})
	if err != nil || string(out) != string([]byte{9, 8, 7}) {
		t.Fatalf("out=%v err=%v", out, err)
	}

	addr, err := c.GetAddress()
	if err != nil || addr == "" {
		t.Fatalf("addr=%q err=%v", addr, err)
	}
}

func TestClient_WaitTimeout(t *testing.T) {
	c := memory.NewClient(memory.ClientConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := c.WaitForTransaction(ctx, "0xnever")
	if !pkgerrors.IsCode(err, web3.CodeTimeout) {
		t.Fatalf("code=%s err=%v", pkgerrors.Code(err), err)
	}
}

func TestClient_CanceledContext(t *testing.T) {
	c := memory.NewClient(memory.ClientConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.GetBalance(ctx, "0x1")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestClient_Instrumented(t *testing.T) {
	ctx := context.Background()
	from := "0x1111111111111111111111111111111111111111"
	raw := memory.NewClient(memory.ClientConfig{
		Address: from,
		InitialBalances: map[string]*big.Int{
			from: big.NewInt(100),
		},
	})
	c := web3.NewInstrumentedClient(raw)
	_, _ = c.GetChainID(ctx)
	_, _ = c.GetBlockNumber(ctx)
	_, _ = c.GetBalance(ctx, from)
	_, _ = c.EstimateGas(ctx, "0x2", nil)
	_, _ = c.CallContract(ctx, "0x2", nil)
	tx, err := c.Transfer(ctx, "0x2222222222222222222222222222222222222222", big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetTransactionReceipt(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.WaitForTransaction(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = c.GetAddress()
	c.Close()
}

func TestStore_AddGetPin(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStore(memory.StoreConfig{GatewayURL: "https://gateway.test"})

	cid, err := s.Add(ctx, []byte("hello"))
	if err != nil || cid == "" {
		t.Fatalf("cid=%q err=%v", cid, err)
	}
	if cid != memory.ContentID([]byte("hello")) {
		t.Fatalf("cid mismatch %q", cid)
	}

	data, err := s.Get(ctx, cid)
	if err != nil || string(data) != "hello" {
		t.Fatalf("data=%q err=%v", data, err)
	}

	type payload struct {
		Name string `json:"name"`
	}
	jsonCID, err := s.AddJSON(ctx, payload{Name: "nft"})
	if err != nil {
		t.Fatal(err)
	}
	var got payload
	if err := s.GetJSON(ctx, jsonCID, &got); err != nil || got.Name != "nft" {
		t.Fatalf("got=%+v err=%v", got, err)
	}

	if err := s.Pin(ctx, cid); err != nil {
		t.Fatal(err)
	}
	pins, err := s.ListPins(ctx)
	if err != nil || len(pins) != 1 || pins[0] != cid {
		t.Fatalf("pins=%v err=%v", pins, err)
	}
	if err := s.Unpin(ctx, cid); err != nil {
		t.Fatal(err)
	}
	pins, _ = s.ListPins(ctx)
	if len(pins) != 0 {
		t.Fatalf("pins after unpin=%v", pins)
	}

	url := s.GetURL(cid)
	if url != "https://gateway.test/ipfs/"+cid {
		t.Fatalf("url=%q", url)
	}

	_, err = s.Get(ctx, "missing")
	if !pkgerrors.IsCode(err, web3.CodeNotFound) {
		t.Fatalf("code=%s", pkgerrors.Code(err))
	}
	if err := s.Pin(ctx, "missing"); !pkgerrors.IsCode(err, web3.CodeNotFound) {
		t.Fatalf("pin missing code=%s", pkgerrors.Code(err))
	}
}

func TestStore_GetJSONInvalid(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStore(memory.StoreConfig{})
	cid, err := s.Add(ctx, []byte("not-json"))
	if err != nil {
		t.Fatal(err)
	}
	var dest map[string]any
	err = s.GetJSON(ctx, cid, &dest)
	if !pkgerrors.IsCode(err, web3.CodeStorageFailed) {
		t.Fatalf("code=%s err=%v", pkgerrors.Code(err), err)
	}
}

func TestStore_Instrumented(t *testing.T) {
	ctx := context.Background()
	raw := memory.NewStore(memory.StoreConfig{})
	s := web3.NewInstrumentedStore(raw)
	cid, err := s.Add(ctx, []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	jsonCID, err := s.AddJSON(ctx, map[string]int{"a": 1})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Get(ctx, cid)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]int
	if err := s.GetJSON(ctx, jsonCID, &m); err != nil || m["a"] != 1 {
		t.Fatalf("m=%v err=%v", m, err)
	}
	if err := s.Pin(ctx, cid); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ListPins(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.Unpin(ctx, cid); err != nil {
		t.Fatal(err)
	}
	_ = s.GetURL(cid)
}

func TestVerifier_MemorySignature(t *testing.T) {
	ctx := context.Background()
	v := memory.NewVerifier()
	msg, err := v.CreateMessage("example.com", "0xAbC", "https://example.com", "hi", 1)
	if err != nil {
		t.Fatal(err)
	}
	sig := memory.MemorySignature(msg)
	ok, err := v.Verify(ctx, msg, sig)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}

	_, err = v.Verify(ctx, msg, sig)
	if !pkgerrors.IsCode(err, web3.CodeNonceReused) {
		t.Fatalf("code=%s", pkgerrors.Code(err))
	}
}

func TestVerifier_RejectBadSigAndTimeBounds(t *testing.T) {
	ctx := context.Background()
	v := memory.NewVerifier()
	msg, err := v.CreateMessage("example.com", "0x1", "https://example.com", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := v.Verify(ctx, msg, "wrong")
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}

	past := time.Now().Add(-time.Minute)
	msg.ExpirationTime = &past
	msg.Nonce = "other"
	_, err = v.Verify(ctx, msg, memory.MemorySignature(msg))
	if !pkgerrors.IsCode(err, web3.CodeMessageExpired) {
		t.Fatalf("code=%s", pkgerrors.Code(err))
	}
}

func TestVerifier_ConcurrentNonce(t *testing.T) {
	v := memory.NewVerifier()
	msg, err := v.CreateMessage("example.com", "0x1", "https://example.com", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	sig := memory.MemorySignature(msg)

	const n = 24
	var wg sync.WaitGroup
	errs := make(chan error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, err := v.Verify(context.Background(), msg, sig)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	var okCount, reuse int
	for err := range errs {
		if err == nil {
			okCount++
		} else if pkgerrors.IsCode(err, web3.CodeNonceReused) {
			reuse++
		} else {
			t.Fatalf("unexpected: %v", err)
		}
	}
	if okCount != 1 || reuse != n-1 {
		t.Fatalf("ok=%d reuse=%d", okCount, reuse)
	}
}

func TestVerifier_Instrumented(t *testing.T) {
	raw := memory.NewVerifier()
	v := web3.NewInstrumentedVerifier(raw)
	msg, err := v.CreateMessage("example.com", "0x1", "https://example.com", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := v.Verify(context.Background(), msg, memory.MemorySignature(msg))
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
