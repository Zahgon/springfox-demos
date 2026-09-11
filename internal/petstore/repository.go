package petstore

import "sync"

// repository is springfox.petstore.repository.MapBackedRepository: a map keyed
// by each value's own identifier.
type repository[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]V
	// order records insertion order so that `where` iterates deterministically.
	order []K
	idOf  func(V) K
}

func newRepository[K comparable, V any](idOf func(V) K) *repository[K, V] {
	return &repository[K, V]{items: map[K]V{}, idOf: idOf}
}

func (r *repository[K, V]) add(v V) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := r.idOf(v)
	if _, ok := r.items[k]; !ok {
		r.order = append(r.order, k)
	}
	r.items[k] = v
}

func (r *repository[K, V]) get(k K) (V, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.items[k]
	return v, ok
}

func (r *repository[K, V]) exists(k K) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.items[k]
	return ok
}

func (r *repository[K, V]) delete(k K) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, k)
	for i, key := range r.order {
		if key == k {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
}

func (r *repository[K, V]) where(pred func(V) bool) []V {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []V{}
	for _, k := range r.order {
		v, ok := r.items[k]
		if ok && pred(v) {
			out = append(out, v)
		}
	}
	return out
}
