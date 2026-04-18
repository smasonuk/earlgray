// Package focus tracks which element has keyboard focus.
package focus

// Manager tracks the currently focused element by ID.
type Manager struct {
	focusedID uintptr
}

// New creates a new focus Manager.
func New() *Manager {
	return &Manager{}
}

// Set moves focus to the element with the given ID.
func (m *Manager) Set(id uintptr) {
	m.focusedID = id
}

// FocusedID returns the ID of the currently focused element, or 0.
func (m *Manager) FocusedID() uintptr {
	return m.focusedID
}

// Clear removes focus.
func (m *Manager) Clear() {
	m.focusedID = 0
}
