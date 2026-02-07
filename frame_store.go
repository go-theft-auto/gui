package gui

import "sync"

// Cleanable is implemented by stores that need frame-based cleanup.
// Each frame, stale entries (not accessed this frame) are removed.
type Cleanable interface {
	Cleanup(currentFrame uint64)
}

// Global registry for automatic cleanup of all FrameStores.
// Uses a mutex for thread-safety during registration.
var (
	registeredStores []Cleanable
	registryMu       sync.Mutex
	currentFrame     uint64
)

// registerStore adds a store to the global cleanup registry.
// Called automatically by NewFrameStore.
func registerStore(store Cleanable) {
	registryMu.Lock()
	registeredStores = append(registeredStores, store)
	registryMu.Unlock()
}

// NextFrame advances the frame counter and cleans all registered stores.
// Call this once at the start of each GUI frame (typically in Context.Reset).
// Stale entries (not accessed in the previous frame) are removed automatically.
func NextFrame() {
	currentFrame++
	registryMu.Lock()
	stores := registeredStores // Copy slice under lock
	registryMu.Unlock()

	for _, store := range stores {
		store.Cleanup(currentFrame)
	}
}

// CurrentFrameCount returns the current frame counter.
// Useful for debugging or advanced use cases.
func CurrentFrameCount() uint64 {
	return currentFrame
}

// stateEntry wraps a state value with frame tracking for staleness detection.
type stateEntry[T any] struct {
	value     T
	lastFrame uint64
}

// FrameStore is a type-safe store for widget state that automatically
// cleans up unused entries each frame.
//
// Unlike the old StateStore which used any (interface{}) and required
// type assertions, FrameStore is fully generic - no runtime type checks,
// no allocations for boxing primitive types.
//
// Usage:
//
//	// At package level - create one store per state type
//	var sectionStore = gui.NewFrameStore[SectionState]()
//
//	// In widget code - get state with compile-time type safety
//	func (ctx *Context) Section(label string) {
//	    id := ctx.GetID(label)
//	    state := sectionStore.Get(id, SectionState{Open: false})
//	    // state is *SectionState - no type assertion needed
//	    state.Open = !state.Open  // Direct modification
//	}
//
// For user-defined widgets, create your own FrameStore without modifying gui:
//
//	var myStore = gui.NewFrameStore[MyWidgetState]()
type FrameStore[T any] struct {
	states map[ID]*stateEntry[T]
	mu     sync.RWMutex // Protects concurrent access
}

// NewFrameStore creates a new type-safe state store and registers it
// for automatic cleanup. The store will automatically remove entries
// that weren't accessed in the previous frame.
//
// Call this at package initialization time (package-level var):
//
//	var sliderStore = gui.NewFrameStore[SliderState]()
func NewFrameStore[T any]() *FrameStore[T] {
	store := &FrameStore[T]{
		states: make(map[ID]*stateEntry[T]),
	}
	registerStore(store)
	return store
}

// Get retrieves state for the given ID, or creates it with defaultVal if not found.
// Returns a pointer to the state, allowing direct modification.
// The state is automatically marked as "used this frame" to prevent cleanup.
//
// This method is safe for concurrent use.
func (s *FrameStore[T]) Get(id ID, defaultVal T) *T {
	s.mu.RLock()
	entry, ok := s.states[id]
	s.mu.RUnlock()

	if ok {
		// Fast path: entry exists, just update frame
		s.mu.Lock()
		entry.lastFrame = currentFrame
		s.mu.Unlock()
		return &entry.value
	}

	// Slow path: create new entry
	s.mu.Lock()
	// Double-check after acquiring write lock
	if entry, ok = s.states[id]; ok {
		entry.lastFrame = currentFrame
		s.mu.Unlock()
		return &entry.value
	}

	entry = &stateEntry[T]{
		value:     defaultVal,
		lastFrame: currentFrame,
	}
	s.states[id] = entry
	s.mu.Unlock()
	return &entry.value
}

// GetIfExists retrieves state only if it already exists.
// Returns nil if no state exists for this ID.
// Does NOT create default state or mark as used.
func (s *FrameStore[T]) GetIfExists(id ID) *T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if entry, ok := s.states[id]; ok {
		return &entry.value
	}
	return nil
}

// Set explicitly sets state for an ID.
// Creates or updates the entry and marks it as used this frame.
func (s *FrameStore[T]) Set(id ID, value T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.states[id]; ok {
		entry.value = value
		entry.lastFrame = currentFrame
	} else {
		s.states[id] = &stateEntry[T]{
			value:     value,
			lastFrame: currentFrame,
		}
	}
}

// Delete explicitly removes state for an ID.
// Use this when you know state is no longer needed (e.g., widget destroyed).
func (s *FrameStore[T]) Delete(id ID) {
	s.mu.Lock()
	delete(s.states, id)
	s.mu.Unlock()
}

// Cleanup removes all entries that weren't accessed in the previous frame.
// This is called automatically by NextFrame() - don't call it manually.
func (s *FrameStore[T]) Cleanup(frame uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove entries not used in the previous frame
	// (frame-1 because we just incremented in NextFrame)
	threshold := frame - 1
	for id, entry := range s.states {
		if entry.lastFrame < threshold {
			delete(s.states, id)
		}
	}
}

// Len returns the number of stored entries.
// Useful for debugging and monitoring.
func (s *FrameStore[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.states)
}

// Clear removes all entries immediately.
// Useful for resetting state (e.g., when switching scenes).
func (s *FrameStore[T]) Clear() {
	s.mu.Lock()
	s.states = make(map[ID]*stateEntry[T])
	s.mu.Unlock()
}
