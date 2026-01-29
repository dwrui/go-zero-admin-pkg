package grand

import (
	"crypto/rand"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gcode"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gerror"
)

const (
	// uint32随机数的缓冲区大小。
	bufferChanSize = 10000
)

// bufferChan 是一个缓冲区，用于存储随机字节，
// 每个项目存储 4 个字节。
var bufferChan = make(chan []byte, bufferChanSize)

func init() {
	go asyncProducingRandomBufferBytesLoop()
}

// asyncProducingRandomBufferBytesLoop 是一个命名的 goroutine，
// 它使用异步 goroutine 来生产随机字节，
// 并使用缓冲区 chan 来存储随机字节。
// 因此，它具有高性能生成随机数的能力。
func asyncProducingRandomBufferBytesLoop() {
	var step int
	for {
		buffer := make([]byte, 1024)
		if n, err := rand.Read(buffer); err != nil {
			panic(gerror.WrapCode(gcode.CodeInternalError, err, `error reading random buffer from system`))
		} else {
			// The random buffer from system is very expensive,
			// 所以通过改变步骤值，
			// 可以充分利用系统提供的随机缓冲区，
			// 从而提高性能。
			// for _, step = range []int{4, 5, 6, 7} {
			for _, step = range []int{4} {
				for i := 0; i <= n-4; i += step {
					bufferChan <- buffer[i : i+4]
				}
			}
		}
	}
}
