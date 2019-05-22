package abi

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/aBitcoinDiamond/slave/dapp/evm/executor/vm/common"
	"github.com/stretchr/testify/assert"
)

func TestABI_Pack(t *testing.T) {
	abiData := `[{"constant":false,"inputs":[],"Name":"kill","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"Name":"newValue","type":"uint256"}],"Name":"set","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":true,"inputs":[],"Name":"get","outputs":[{"Name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]`

	for _, test := range []struct {
		input  string
		output string
	}{
		{
			"get()",
			"0x6d4ce63c",
		},
		{
			"kill()",
			"0x41c0e1b5",
		},
		{
			"set(100)",
			"0x60fe47b10000000000000000000000000000000000000000000000000000000000000064",
		},
		{
			`set("100")`,
			"0x60fe47b10000000000000000000000000000000000000000000000000000000000000064",
		},
	} {
		_, data, err := Pack(test.input, abiData, false)
		assert.NoError(t, err)
		assert.EqualValues(t, test.output, common.Bytes2Hex(data))
	}
}

func TestABI_Unpack(t *testing.T) {
	abiData := `[{"constant":false,"inputs":[],"Name":"get1","outputs":[{"Name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"Name":"f","outputs":[{"Name":"","type":"uint256"},{"Name":"","type":"bool"},{"Name":"","type":"uint256"}],"payable":false,"stateMutability":"pure","type":"function"},{"constant":true,"inputs":[{"Name":"_a","type":"uint256"},{"Name":"_b","type":"uint256"}],"Name":"arithmetics","outputs":[{"Name":"o_sum","type":"uint256"},{"Name":"o_product","type":"uint256"}],"payable":false,"stateMutability":"pure","type":"function"},{"constant":true,"inputs":[{"Name":"index","type":"uint256"}],"Name":"getUser","outputs":[{"Name":"","type":"uint256"},{"Name":"","type":"string"},{"Name":"","type":"string"},{"Name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[],"Name":"get2","outputs":[{"Name":"_winningProposal","type":"uint8"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`

	for _, test := range []struct {
		method string
		input  string
		output string
	}{
		{
			"get1",
			"0x0000000000000000000000000000000000000000000000000000000000000021",
			`[{"name":"","type":"uint256","value":33}]`,
		},
		{
			"get2",
			"0x0000000000000000000000000000000000000000000000000000000000000021",
			`[{"name":"_winningProposal","type":"uint8","value":33}]`,
		},
		{
			"arithmetics",
			"0x000000000000000000000000000000000000000000000000000000000000002100000000000000000000000000000000000000000000000000000000000000f2",
			`[{"name":"o_sum","type":"uint256","value":33},{"name":"o_product","type":"uint256","value":242}]`,
		},
		{
			"f",
			"0x000000000000000000000000000000000000000000000000000000000000000700000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
			`[{"name":"","type":"uint256","value":7},{"name":"","type":"bool","value":true},{"name":"","type":"uint256","value":2}]`,
		},
		{
			"getUser",
			"0x0000000000000000000000000000000000000000000000000000000000000021000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000032dcd5000000000000000000000000000000000000000000000000000000000000000a75736572732e6e616d6500000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001175736572732e7573657241646472657373000000000000000000000000000000",
			`[{"name":"","type":"uint256","value":33},{"name":"","type":"string","value":"users.name"},{"name":"","type":"string","value":"users.userAddress"},{"name":"","type":"uint256","value":3333333}]`,
		},
	} {

		data, err := Unpack(common.FromHex(test.input), test.method, abiData)
		assert.NoError(t, err)
		assert.EqualValues(t, test.output, data)
	}
}

func TestProcFuncCall(t *testing.T) {
	for _, test := range []struct {
		input    string
		funcName string
		params   []string
	}{
		{
			"foo(1,2,3)",
			"foo",
			[]string{"1", "2", "3"},
		},
		{
			"foo()",
			"foo",
			nil,
		},
		{
			`foo("a",1,[1,2,3])`,
			"foo",
			[]string{"a", "1", "[1,2,3]"},
		},
	} {
		fn, res, err := procFuncCall(test.input)
		assert.NoError(t, err, "process array string error")
		assert.EqualValues(t, test.funcName, fn, "parse array string error")
		assert.EqualValues(t, test.params, res, "parse array string error")
	}
}

// TestProcArray 测试根据ABI解析数组结构的逻辑
func TestProcArray(t *testing.T) {
	for _, test := range []struct {
		input  string
		output []string
	}{
		{
			"[1,2,3]",
			[]string{"1", "2", "3"},
		},
		{
			"[[1,2,3],[4,5]]",
			[]string{"[1,2,3]", "[4,5]"},
		},
		{
			`["abc","def",  "x,y, z"]`,
			[]string{"abc", "def", "x,y, z"},
		},
		{
			`["[1,2,3]"]`,
			[]string{"[1,2,3]"},
		},
	} {
		res, err := procArrayItem(test.input)
		assert.NoError(t, err, "process array string error")

		assert.EqualValues(t, test.output, res, "parse array string error")
	}
}

// Test_GoValue 测试从字符串依据ABI类型定义，转换为正确的类型并编码的逻辑
func Test_GoValue(t *testing.T) {
	for i, test := range []struct {
		typ    string
		input  string
		output []byte
	}{
		{
			"uint8",
			"2",
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint8[]",
			"[1, 2]",
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint16",
			"2",
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint16[]",
			"[1, 2]",
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint32",
			"2",
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint32[]",
			"[1, 2]",
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint64",
			"2",
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint64[]",
			"[1,2]",
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint256",
			"2",
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint256[]",
			"[1,2]",
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int8",
			"2",
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int8[]",
			"[1,2]",
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int16",
			"2",
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int16[]",
			"[1,2]",
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int32",
			"2",
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int32[]",
			"[1,2]",
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int64",
			"2",
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int64[]",
			"[1,2]",
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int256",
			"2",
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int256[]",
			"[1,2]",
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"bytes1",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes2",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes3",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes4",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes5",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes6",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes7",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes8",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes9",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes10",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes11",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes12",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes13",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes14",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes15",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes16",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes17",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes18",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes19",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes20",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes21",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes22",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes23",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes24",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes24",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes25",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes26",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes27",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes28",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes29",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes30",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes31",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes32",
			"0x01",
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"uint32[2][3][4]",
			"[[[1, 2], [3, 4], [5, 6]], [[7, 8], [9, 10], [11, 12]], [[13, 14], [15, 16], [17, 18]], [[19, 20], [21, 22], [23, 24]]]",
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000700000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000009000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000b000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000000d000000000000000000000000000000000000000000000000000000000000000e000000000000000000000000000000000000000000000000000000000000000f000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000110000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000001300000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000015000000000000000000000000000000000000000000000000000000000000001600000000000000000000000000000000000000000000000000000000000000170000000000000000000000000000000000000000000000000000000000000018"),
		},
		{
			"address",
			common.Hash160Address{1}.ToAddress().String(),
			common.Hex2Bytes("0000000000000000000000000100000000000000000000000000000000000000"),
		},
		{
			"address[]",
			fmt.Sprintf("[%v,%v]", common.Hash160Address{1}.ToAddress(), common.Hash160Address{2}.ToAddress()),
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000001000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000"),
		},
		{
			"bytes32[]",
			//[]common.Hash{{1}, {2}},
			"[0x01, 0x02]",
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000201000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"string",
			"foobar",
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000006666f6f6261720000000000000000000000000000000000000000000000000000"),
		},
	} {
		typ, err := NewType(test.typ)
		if err != nil {
			t.Fatalf("%v failed. Unexpected new type error: %v", i, err)
		}

		value, err := str2GoValue(typ, test.input)
		if err != nil {
			t.Fatalf("%v failed. Unexpected parse go Value error: %v", i, err)
		}

		output, err := typ.pack(reflect.ValueOf(value))
		if err != nil {
			t.Fatalf("%v failed. Unexpected pack error: %v", i, err)
		}

		if !bytes.Equal(output, test.output) {
			t.Errorf("%d failed. Expected bytes: '%x' Got: '%x'", i, test.output, output)
		}
	}
}
