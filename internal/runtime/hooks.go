package runtime

// renderingInstance is set to the current instance being rendered.
// This is safe because rendering is single-threaded.
var renderingInstance *Instance

// IsFocused reports whether any immediate child of the rendering component
// instance is the currently focused node. Call this from a component function
// to know whether to render in a focused visual state.
func IsFocused() bool {
	inst := renderingInstance
	if inst == nil {
		return false
	}
	rt := inst.runtime
	if rt == nil || rt.focused == nil {
		return false
	}
	for _, child := range inst.children {
		if child == rt.focused {
			return true
		}
	}
	return false
}

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
		inst.runtime.MarkDirty()
	}

	return val, setter
}
