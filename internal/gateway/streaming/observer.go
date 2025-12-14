package streaming

// StreamObserver 定义 streaming 生命周期的观察者接口
type StreamObserver interface {
	// OnFirstChunk 在首个 chunk 成功写出并 flush 后调用
	OnFirstChunk(ctx *StreamingContext) error

	// OnChunk 在每个 chunk 交付后调用
	// 返回 (stop, err)：
	// - stop=true 表示应该中断流
	// - err 表示观察者内部错误
	OnChunk(ctx *StreamingContext, delta *Delta) (stop bool, err error)

	// OnEnd 在流结束时调用（无论正常结束还是异常）
	OnEnd(ctx *StreamingContext) error
}

// ObserverGroup 串联多个 observer，依次调用
type ObserverGroup struct {
	observers []StreamObserver
}

// NewObserverGroup 创建观察者组
func NewObserverGroup(observers ...StreamObserver) *ObserverGroup {
	return &ObserverGroup{
		observers: observers,
	}
}

// OnFirstChunk 依次调用所有 observer 的 OnFirstChunk
func (og *ObserverGroup) OnFirstChunk(ctx *StreamingContext) error {
	for _, obs := range og.observers {
		if err := obs.OnFirstChunk(ctx); err != nil {
			return err
		}
	}
	return nil
}

// OnChunk 依次调用所有 observer 的 OnChunk
// 如果任何 observer 返回 stop=true，立即停止并返回
func (og *ObserverGroup) OnChunk(ctx *StreamingContext, delta *Delta) (stop bool, err error) {
	for _, obs := range og.observers {
		stop, err := obs.OnChunk(ctx, delta)
		if stop || err != nil {
			return stop, err
		}
	}
	return false, nil
}

// OnEnd 依次调用所有 observer 的 OnEnd
// 继续调用所有 observer，即使某个出错（但记录第一个错误）
func (og *ObserverGroup) OnEnd(ctx *StreamingContext) error {
	var firstErr error
	for _, obs := range og.observers {
		if err := obs.OnEnd(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
