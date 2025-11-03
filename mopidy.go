package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type MopidyRPCRequest struct {
	Method  string                 `json:"method"`
	JSONRpc string                 `json:"jsonrpc"`
	Params  map[string]interface{} `json:"params,omitempty"`
	ID      int                    `json:"id"`
}

type MopidyRPCResponse struct {
	JSONRpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func mopidyRPCRequest(server *MopidyServer, method string, params map[string]interface{}) (interface{}, error) {
	req := MopidyRPCRequest{
		Method:  method,
		JSONRpc: "2.0",
		Params:  params,
		ID:      1,
	}

	if req.Params == nil {
		req.Params = make(map[string]interface{})
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("http://%s:%d/mopidy/rpc", server.Host, server.Port)
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	var resp MopidyRPCResponse
	err = json.Unmarshal(respBody, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("mopidy error: %s", resp.Error.Message)
	}

	return resp.Result, nil
}
