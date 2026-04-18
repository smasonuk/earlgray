package runtime

// renderingInstance is set to the current instance being rendered.
// This is safe because rendering is single-threaded.
var renderingInstance *Instance

// UseState implements React-like state hooks.
// T must match the type used when the hook slot was first initialized.
func UseState[T any](initial T) (T, func(T)) {
	if renderingInstance == nil {
		panic("tui: UseState called outside component render")
	}
	inst := renderingInstance
	idx := inst.hookIdx
	inst.hookIdx++

	if idx >= len(inst.hookSlots) {
		inst.hookSlots = append(inst.hookSlots, initial)
	}

	val := inst.hookSlots[idx].(T)

	setter := func(v T) {
		inst.hookSlots[idx] = v
		if globalRuntime != nil {
			globalRuntime.MarkDirty()
		}
	}

	return val, setter
}

// globalRuntime holds the running Runtime so UseState can trigger redraws.
// Set by Runtime.Run.
var globalRuntime *Runtime
