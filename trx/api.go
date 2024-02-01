package trx

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"math/big"
	"net/http"
)

type TrxApiClient struct {
	BaseUrl string
	Client  *http.Client
	Log     *slog.Logger
}

func NewTrxApiClient(baseUrl string, log *slog.Logger) *TrxApiClient {
	return &TrxApiClient{
		Client:  &http.Client{},
		BaseUrl: baseUrl,
		Log:     log,
	}
}

func (c *TrxApiClient) DoPost(ctx context.Context, path string, req any, res any) error {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return err
	}
	c.Log.Debug("ReqPOST", "path", path, "req", string(reqBody))
	payload := bytes.NewBuffer(reqBody)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseUrl+path, payload)
	if err != nil {
		return err
	}

	httpReq.Header.Add("accept", "application/json")
	httpReq.Header.Add("content-type", "application/json")

	httpRes, err := c.Client.Do(httpReq)
	if err != nil {
		return err
	}

	defer httpRes.Body.Close()
	body, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return err
	}

	c.Log.Debug("ResPOST", "path", path, "res", string(body))

	return json.Unmarshal(body, res)
}

type CreateTransactionRequest struct {
	OwnerAddress string `json:"owner_address"`
	ToAddress    string `json:"to_address"`
	Amount       uint64 `json:"amount"`
	Visible      bool   `json:"visible"`
}

type CreateTransactionResponse struct {
	TxId       string          `json:"txID"`
	RawData    json.RawMessage `json:"raw_data"`
	RawDataHex string          `json:"raw_data_hex"`
	Error      string          `json:"error"`
}

func (c *TrxApiClient) CreateTransaction(ctx context.Context, req CreateTransactionRequest) (CreateTransactionResponse, error) {
	var res CreateTransactionResponse
	err := c.DoPost(ctx, "/wallet/createtransaction", req, &res)
	return res, err
}

type RawTxData struct {
	Contract      json.RawMessage `json:"contract,omitempty"`
	RefBlockBytes json.RawMessage `json:"ref_block_bytes,omitempty"`
	RefBlockHash  json.RawMessage `json:"ref_block_hash,omitempty"`
	Expiration    json.RawMessage `json:"expiration,omitempty"`
	FeeLimit      uint64          `json:"fee_limit,omitempty"`
	Timestamp     json.RawMessage `json:"timestamp,omitempty"`
}

type TriggerContractRequest struct {
	OwnerAddress    string `json:"owner_address"`
	ContractAddress string `json:"contract_address"`
	FeeLimit        uint64 `json:"fee_limit"`
	//FunctionSelector string `json:"function_selector"`
	CallValue uint64 `json:"call_value"`
	Data      string `json:"data"`
	Visible   bool   `json:"visible"`
}

type TriggerSmartContractResponse struct {
	Result struct {
		Result  bool   `json:"result"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"result"`
	Transaction struct {
		TxId       string          `json:"txID"`
		RawData    json.RawMessage `json:"raw_data"`
		RawDataHex string          `json:"raw_data_hex"`
	} `json:"transaction"`
}

type TriggerConstantContractResponse struct {
	Result struct {
		Result  bool   `json:"result"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"result"`
	EnergyUsed  uint64 `json:"energy_used"`
	Transaction struct {
		TxId       string    `json:"txID"`
		RawData    RawTxData `json:"raw_data"`
		RawDataHex string    `json:"raw_data_hex"`
	} `json:"transaction"`
}

func (c *TrxApiClient) TriggerConstantContract(ctx context.Context, req TriggerContractRequest) (TriggerConstantContractResponse, error) {
	var res TriggerConstantContractResponse
	err := c.DoPost(ctx, "/wallet/triggerconstantcontract", req, &res)
	return res, err
}

func (c *TrxApiClient) TriggerSmartContract(ctx context.Context, req TriggerContractRequest) (TriggerSmartContractResponse, error) {
	var res TriggerSmartContractResponse
	err := c.DoPost(ctx, "/wallet/triggersmartcontract", req, &res)
	return res, err
}

type BroadcastTransactionRequest struct {
	//TxId    string          `json:"txID"`
	Visible   bool            `json:"visible"`
	RawData   json.RawMessage `json:"raw_data"`
	Signature []string        `json:"signature"`
}

type BroadcastTransactionResponse struct {
	Result  bool   `json:"result"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (c *TrxApiClient) BroadcastTransaction(ctx context.Context, req BroadcastTransactionRequest) (BroadcastTransactionResponse, error) {
	var res BroadcastTransactionResponse
	err := c.DoPost(ctx, "/wallet/broadcasttransaction", req, &res)
	return res, err
}

type ChainParameters struct {
	BandwidthPrice *big.Int
	EnergyPrice    *big.Int
}

func (c *TrxApiClient) GetChainParameters(ctx context.Context) (ChainParameters, error) {
	var res ChainParameters
	type parameterKeyValue struct {
		Key   string `json:"key"`
		Value int64  `json:"value"`
	}
	type chainParametersResponse struct {
		Result []parameterKeyValue `json:"chainParameter"`
	}
	var internal chainParametersResponse
	err := c.DoPost(ctx, "/wallet/getchainparameters", nil, &internal)
	if err != nil {
		return res, err
	}
	for _, kv := range internal.Result {
		switch kv.Key {
		case "getTransactionFee":
			res.BandwidthPrice = big.NewInt(kv.Value)
		case "getEnergyFee":
			res.EnergyPrice = big.NewInt(kv.Value)
		}
	}
	return res, err
}
