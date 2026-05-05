package gcoding

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dwrui/go-zero-admin-pkg/utils/tools/grand"
)

/*
*
生成随机码方法
订单号
GenerateOrderNo "PO" PO20250117000001
设备编号
GenerateDeviceCode "EQ" EQ2025010001
合同编号
GenerateContractNo "HT" HT00001
批次编号
GenerateBatchNo "BN"BN20250117001
生成自定义编号
GenerateCode "CG", "20060102", 6 CG20250117000001
拼接参数
GenerateCodeWithParams "PO", "SH" POSH202501170001
自定义编号
GenerateCustomCode "PO", "-", 6 PO-20250117-000001
*/
var (
	codeCounter int64 = 0
	codeMutex   sync.Mutex
)

type CodeGenerator struct {
	Prefix     string
	DateLayout string
	SerialLen  int
	ResetDaily bool
	lastDate   string
}

func NewCodeGenerator(prefix string, options ...CodeOption) *CodeGenerator {
	gen := &CodeGenerator{
		Prefix:     prefix,
		DateLayout: "20060102",
		SerialLen:  4,
		ResetDaily: true,
	}
	for _, opt := range options {
		opt(gen)
	}
	return gen
}

type CodeOption func(*CodeGenerator)

func WithDateLayout(layout string) CodeOption {
	return func(g *CodeGenerator) {
		g.DateLayout = layout
	}
}

func WithSerialLen(length int) CodeOption {
	return func(g *CodeGenerator) {
		g.SerialLen = length
	}
}

func WithResetDaily(reset bool) CodeOption {
	return func(g *CodeGenerator) {
		g.ResetDaily = reset
	}
}

func (g *CodeGenerator) Generate() string {
	return g.GenerateWithSerial(0)
}

func (g *CodeGenerator) GenerateWithSerial(serial int64) string {
	now := time.Now()
	dateStr := ""
	if g.DateLayout != "" {
		dateStr = now.Format(g.DateLayout)
	}

	codeMutex.Lock()
	defer codeMutex.Unlock()

	if g.ResetDaily && g.lastDate != dateStr {
		codeCounter = 0
		g.lastDate = dateStr
	}

	if serial > 0 {
		codeCounter = serial
	} else {
		codeCounter++
	}

	serialStr := fmt.Sprintf("%0*d", g.SerialLen, codeCounter)

	var parts []string
	if g.Prefix != "" {
		parts = append(parts, g.Prefix)
	}
	if dateStr != "" {
		parts = append(parts, dateStr)
	}
	parts = append(parts, serialStr)

	return strings.Join(parts, "")
}

func (g *CodeGenerator) GenerateWithTime() string {
	now := time.Now()
	timeStr := now.Format("150405")

	codeMutex.Lock()
	defer codeMutex.Unlock()

	codeCounter++

	serialStr := fmt.Sprintf("%0*d", g.SerialLen, codeCounter)

	var parts []string
	if g.Prefix != "" {
		parts = append(parts, g.Prefix)
	}
	parts = append(parts, timeStr)
	parts = append(parts, serialStr)

	return strings.Join(parts, "")
}

func (g *CodeGenerator) GenerateWithRandom() string {
	now := time.Now()
	dateStr := ""
	if g.DateLayout != "" {
		dateStr = now.Format(g.DateLayout)
	}

	randomStr := grand.S(g.SerialLen)

	var parts []string
	if g.Prefix != "" {
		parts = append(parts, g.Prefix)
	}
	if dateStr != "" {
		parts = append(parts, dateStr)
	}
	parts = append(parts, randomStr)

	return strings.Join(parts, "")
}

func GenerateOrderNo(prefix string) string {
	gen := NewCodeGenerator(prefix, WithSerialLen(6))
	return gen.Generate()
}

func GenerateDeviceCode(prefix string) string {
	gen := NewCodeGenerator(prefix, WithDateLayout("200601"), WithSerialLen(4))
	return gen.Generate()
}

func GenerateContractNo(prefix string) string {
	gen := NewCodeGenerator(prefix, WithSerialLen(5))
	return gen.Generate()
}

func GenerateBatchNo(prefix string) string {
	gen := NewCodeGenerator(prefix, WithDateLayout("20060102"), WithSerialLen(3))
	return gen.Generate()
}

func GenerateCode(prefix string, dateLayout string, serialLen int) string {
	gen := NewCodeGenerator(prefix, WithDateLayout(dateLayout), WithSerialLen(serialLen))
	return gen.Generate()
}

func GenerateCodeWithParams(params ...string) string {
	now := time.Now()
	dateStr := now.Format("20060102")

	codeMutex.Lock()
	codeCounter++
	counter := codeCounter
	codeMutex.Unlock()

	serialStr := fmt.Sprintf("%04d", counter)

	var parts []string
	for _, p := range params {
		if p != "" {
			parts = append(parts, p)
		}
	}
	parts = append(parts, dateStr, serialStr)

	return strings.Join(parts, "")
}

func GenerateCustomCode(prefix string, middle string, serialLen int) string {
	now := time.Now()
	dateStr := now.Format("20060102")

	codeMutex.Lock()
	codeCounter++
	counter := codeCounter
	codeMutex.Unlock()

	serialStr := fmt.Sprintf("%0*d", serialLen, counter)

	return fmt.Sprintf("%s%s%s%s", prefix, middle, dateStr, serialStr)
}
