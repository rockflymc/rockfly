package event

// Cancellable is implemented by any event or event context that can be cancelled.
type Cancellable interface {
	Cancelled() bool
	Cancel()
}

// ResettableCancellable is implemented by events whose cancellation state can be explicitly set or revoked.
type ResettableCancellable interface {
	Cancellable
	Uncancel()
	SetCancelled(cancelled bool)
}

// Context represents the context of an event. Handlers of an event may call methods on the context to change
// the result of the event.
type Context[T any] struct {
	cancel bool
	val    T
}

// C returns a new event context.
func C[T any](v T) *Context[T] {
	return &Context[T]{val: v}
}

// Val returns the subject of the Context.
func (ctx *Context[T]) Val() T {
	return ctx.val
}

// Cancelled returns whether the context has been cancelled.
func (ctx *Context[T]) Cancelled() bool {
	return ctx.cancel
}

// Cancel cancels the context.
func (ctx *Context[T]) Cancel() {
	ctx.cancel = true
}

// Uncancel revokes previous cancellation of the context.
func (ctx *Context[T]) Uncancel() {
	ctx.cancel = false
}

// SetCancelled explicitly sets the cancellation status.
func (ctx *Context[T]) SetCancelled(cancelled bool) {
	ctx.cancel = cancelled
}
