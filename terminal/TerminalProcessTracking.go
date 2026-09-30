package terminal

import (
	"context"
	"sync"
)

// Global process tracking map to store active context cancellation functions.
var (
	ProcMutex sync.Mutex
	ProcPool  = make(map[int]context.CancelFunc)
	NextID    = 1000 // Arbitrary starting index to separate from standard build IDs
)
