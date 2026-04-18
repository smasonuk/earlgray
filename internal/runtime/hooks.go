package runtime

import "reflect"

// renderingInstance is set to the current instance being rendered.
// This is safe because rendering is single-threaded.
var renderingInstance *Instance

// containsInstance reports whether the target instance is in the subtree rooted at root.
func containsInstance(root, target *Instance) bool {
	if root == target {
		return true
	}
	for _, child := range root.children {
		if containsInstance(child, target) {
			return true
		}
	}
	return false
}

// IsFocused reports whether the current component's rendered subtree contains
// the currently focused node. Call this from a component function to know
// whether to render in a focused visual state.
func IsFocused() bool {
	inst := renderingInstance
	if inst == nil {
		return false
	}
	rt := inst.runtime
	if rt == nil || rt.focused == nil {
		return false
	}
	return containsInstance(inst, rt.focused)
}

// UseState implements React-like state hooks.
// T must match the type used when the hook slot was first initialized.
func UseState[T any](initial T) (T, func(T)) {
	inst, idx, val := bindStateHook(initial)

	setter := func(v T) {
		enqueueStateUpdate(inst, idx, v, nil)
	}

	return val, setter
}

// UseStateGuarded is an internal variant of UseState that can drop queued
// updates if a guard fails when the update is applied on the app loop.
func UseStateGuarded[T any](initial T) (T, func(T), func(T, func() bool)) {
	inst, idx, val := bindStateHook(initial)

	setter := func(v T) {
		enqueueStateUpdate(inst, idx, v, nil)
	}
	guardedSetter := func(v T, guard func() bool) {
		enqueueStateUpdate(inst, idx, v, guard)
	}

	return val, setter, guardedSetter
}

func bindStateHook[T any](initial T) (*Instance, int, T) {
	if renderingInstance == nil {
		panic("tui: UseState called outside component render")
	}
	inst := renderingInstance
	idx := inst.hookIdx
	inst.hookIdx++

	if idx >= len(inst.hookSlots) {
		inst.hookSlots = append(inst.hookSlots, hookSlot{
			kind:  hookState,
			state: initial,
		})
	}

	slot := &inst.hookSlots[idx]
	if slot.kind != hookState {
		panic("tui: hook order changed: UseState called where another hook type was previously used")
	}

	val := slot.state.(T)
	return inst, idx, val
}

func enqueueStateUpdate[T any](inst *Instance, idx int, v T, guard func() bool) {
	rt := inst.runtime
	if rt == nil {
		return
	}
	rt.enqueue(func() {
		if !isInstanceMounted(rt.root, inst) {
			return
		}
		if guard != nil && !guard() {
			return
		}
		if idx < 0 || idx >= len(inst.hookSlots) {
			return
		}
		slot := &inst.hookSlots[idx]
		if slot.kind != hookState {
			return
		}
		slot.state = v
		rt.MarkDirty()
	})
}

// UseEffect registers a component-local side effect.
func UseEffect(effect func() func(), deps ...any) {
	if renderingInstance == nil {
		panic("tui: UseEffect called outside component render")
	}

	inst := renderingInstance
	idx := inst.hookIdx
	inst.hookIdx++

	if effect == nil {
		effect = func() func() { return nil }
	}

	nextDeps := copyDeps(deps)

	if idx >= len(inst.hookSlots) {
		inst.hookSlots = append(inst.hookSlots, hookSlot{
			kind: hookEffect,
			effect: effectSlot{
				deps: nextDeps,
			},
		})
		inst.runtime.enqueueEffect(inst, idx, effect)
		return
	}

	slot := &inst.hookSlots[idx]
	if slot.kind != hookEffect {
		panic("tui: hook order changed: UseEffect called where another hook type was previously used")
	}

	if depsChanged(slot.effect.deps, nextDeps) {
		slot.effect.deps = nextDeps
		inst.runtime.enqueueEffect(inst, idx, effect)
	}
}

// UseRef returns a stable pointer for component-local mutable state that does
// not itself trigger rerenders when mutated.
func UseRef[T any](initial T) *T {
	if renderingInstance == nil {
		panic("tui: UseRef called outside component render")
	}

	inst := renderingInstance
	idx := inst.hookIdx
	inst.hookIdx++

	if idx >= len(inst.hookSlots) {
		v := new(T)
		*v = initial
		inst.hookSlots = append(inst.hookSlots, hookSlot{
			kind:  hookRef,
			state: v,
		})
	}

	slot := &inst.hookSlots[idx]
	if slot.kind != hookRef {
		panic("tui: hook order changed: UseRef called where another hook type was previously used")
	}

	return slot.state.(*T)
}

func copyDeps(deps []any) []any {
	if len(deps) == 0 {
		return nil
	}
	out := make([]any, len(deps))
	copy(out, deps)
	return out
}

func depsChanged(prev, next []any) bool {
	if len(prev) != len(next) {
		return true
	}
	for i := range prev {
		if !reflect.DeepEqual(prev[i], next[i]) {
			return true
		}
	}
	return false
}
