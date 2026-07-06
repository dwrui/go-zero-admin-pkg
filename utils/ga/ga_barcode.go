package ga

import (
	"fmt"
	"strconv"
	"time"

	"github.com/dwrui/go-zero-admin-pkg/utils/tools/grand"
)

const (
	BarcodeDomainMaterial = "M"
	BarcodeDomainGoods    = "G"

	BarcodeFormatEAN13      int64 = 1
	BarcodeFormatCode128    int64 = 2
	BarcodeFormatQR         int64 = 3
	BarcodeFormatEAN8       int64 = 4
	BarcodeFormatDatamatrix int64 = 5
)

type BarcodeGenOptions struct {
	Domain      string // M=物料 G=商品
	BusinessId  int64
	OwnerId     int64 // material_id 或 goods_id
	BarcodeType int64
	CodeFormat  int64
	Seq         int64
}

func GenerateBarcode(opts BarcodeGenOptions) string {
	switch opts.CodeFormat {
	case BarcodeFormatEAN13:
		return generateEAN13(opts)
	case BarcodeFormatEAN8:
		return generateEAN8(opts)
	case BarcodeFormatCode128:
		return generateCode128(opts)
	case BarcodeFormatQR:
		return generateQR(opts)
	case BarcodeFormatDatamatrix:
		return generateDatamatrix(opts)
	default:
		return generateCode128(opts)
	}
}

func generateEAN13(opts BarcodeGenOptions) string {
	prefix := "690"
	if opts.Domain == BarcodeDomainGoods {
		prefix = "691"
	}
	body := fmt.Sprintf("%s%06d%03d", prefix, opts.OwnerId%1000000, opts.Seq%1000)
	return body + strconv.Itoa(EANCheckDigit(body))
}

func generateEAN8(opts BarcodeGenOptions) string {
	body := fmt.Sprintf("%06d%01d", (opts.OwnerId+opts.Seq)%1000000, opts.BarcodeType%10)
	if len(body) > 7 {
		body = body[len(body)-7:]
	}
	return body + strconv.Itoa(EANCheckDigit(body))
}

func generateCode128(opts BarcodeGenOptions) string {
	prefix := "BC"
	if opts.Domain == BarcodeDomainGoods {
		prefix = "GC"
	}
	return fmt.Sprintf("%s%08d%06d", prefix, opts.OwnerId%100000000, opts.Seq%1000000)
}

func generateQR(opts BarcodeGenOptions) string {
	tag := "MAT"
	if opts.Domain == BarcodeDomainGoods {
		tag = "GDS"
	}
	return fmt.Sprintf("%s:%d:%d:%d:%d:%s", tag, opts.BusinessId, opts.OwnerId, opts.BarcodeType, opts.Seq, time.Now().Format("20060102150405"))
}

func generateDatamatrix(opts BarcodeGenOptions) string {
	prefix := "DM"
	if opts.Domain == BarcodeDomainGoods {
		prefix = "DG"
	}
	return fmt.Sprintf("%s%d%06d%s", prefix, opts.OwnerId%100000000, opts.Seq%1000000, grand.S(4))
}

func EANCheckDigit(code string) int {
	sum := 0
	n := len(code)
	for i := 0; i < n; i++ {
		d := int(code[n-1-i] - '0')
		if i%2 == 0 {
			sum += d * 3
		} else {
			sum += d
		}
	}
	return (10 - sum%10) % 10
}
