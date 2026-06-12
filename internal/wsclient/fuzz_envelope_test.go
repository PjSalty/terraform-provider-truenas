package wsclient

import (
	"encoding/json"
	"testing"
)

// FuzzRPCResponse_Unmarshal fuzzes the JSON-RPC envelope parser the
// wsclient feeds with every wire frame. A panic here would crash the
// provider on a malformed server response, which can come from a
// proxy in the path, a partial Cilium drop, or a TrueNAS bug.
//
// The seed corpus seeds the engine with well-formed frames, frames
// with surprising shapes (id-as-string, result-as-array,
// error-as-string, missing-jsonrpc-version), and frankly malformed
// bytes. f.Fuzz mutates these so the engine explores far beyond the
// seeds.
func FuzzRPCResponse_Unmarshal(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{"jsonrpc":"2.0","id":1,"result":"pong"}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"result":null}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"result":[]}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid request"}}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"Internal error","data":{"x":1}}}`),
		[]byte(`{"jsonrpc":"2.0","id":18446744073709551615,"result":null}`),
		[]byte(`{"jsonrpc":"2.0","id":0,"result":null}`),
		[]byte(`{"jsonrpc":"2.0","id":-1,"result":null}`),
		[]byte(`{"jsonrpc":"1.0","id":1,"result":"different version"}`),
		[]byte(`{"result":"missing-version"}`),
		[]byte(`{"jsonrpc":"2.0"}`),
		// surprising shapes that the parser may or may not survive
		[]byte(`{"jsonrpc":"2.0","id":"string-id","result":null}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"result":"a","error":null}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"result":null,"error":{"code":"not-int","message":"x"}}`),
		// malformed bytes
		[]byte(`{`),
		[]byte(`{"jsonrpc":}`),
		[]byte(`null`),
		[]byte(`[]`),
		{0x00, 0x01, 0x02},
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(_ *testing.T, data []byte) {
		var r rpcResponse
		_ = json.Unmarshal(data, &r)
		// Re-marshal whatever survived parsing, round-trip must not panic
		// even if the resulting JSON is structurally different.
		_, _ = json.Marshal(&r)
	})
}

// FuzzRPCError_Unmarshal fuzzes the error variant alone. RPCError
// participates in errors.Is/As chains, so a malformed shape that
// leaves it half-initialized would mask error-class detection in
// the calling code.
func FuzzRPCError_Unmarshal(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{"code":-32600,"message":"Invalid request"}`),
		[]byte(`{"code":0,"message":""}`),
		[]byte(`{"code":-1,"message":"err","data":null}`),
		[]byte(`{"code":-32000,"message":"Server error","data":{"trace":"..."}}`),
		[]byte(`{"code":-32000,"message":"err","data":"string-data"}`),
		[]byte(`{"code":-32000,"message":"err","data":[1,2,3]}`),
		[]byte(`{}`),
		[]byte(`null`),
		// missing required-by-spec fields
		[]byte(`{"message":"no code"}`),
		[]byte(`{"code":-32600}`),
		// malformed
		[]byte(`{"code":}`),
		{0x00, 0x01},
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(_ *testing.T, data []byte) {
		var e RPCError
		_ = json.Unmarshal(data, &e)
		_, _ = json.Marshal(&e)
		_ = e.Error() // String() method must survive any shape too
	})
}

// FuzzJobUpdate_Unmarshal targets the job-progress envelope that
// CallJob polls. Two RawMessage fields make this an easy place for
// shape drift to slip through.
func FuzzJobUpdate_Unmarshal(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{"id":1,"state":"RUNNING","progress":{"percent":50,"description":"hashing"},"result":null}`),
		[]byte(`{"id":1,"state":"SUCCESS","progress":null,"result":{"key":"value"}}`),
		[]byte(`{"id":1,"state":"FAILED","error":"out of disk"}`),
		[]byte(`{"id":1,"state":"WAITING"}`),
		[]byte(`{"id":1,"state":"ABORTED","abort_reason":"user cancelled"}`),
		[]byte(`{"id":-1,"state":""}`),
		[]byte(`{}`),
		[]byte(`{"id":1,"progress":"not-an-object","result":[1,2,3]}`),
		[]byte(`{`),
		{0x00},
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(_ *testing.T, data []byte) {
		var ju jobInfo
		_ = json.Unmarshal(data, &ju)
		_, _ = json.Marshal(&ju)
	})
}
