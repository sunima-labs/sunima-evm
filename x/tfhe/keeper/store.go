package keeper

// KVStore is the minimal key-value store interface the keeper depends on.
//
// It is a deliberate subset of cosmossdk.io/store/types.KVStore — small enough
// to back with an in-memory map for unit tests, large enough to map directly
// onto the real Cosmos SDK store when the module is wired into the chain
// (Stage 5.1 Week 3).
//
// Iteration semantics follow the Cosmos store contract: ranges are
// half-open (start inclusive, end exclusive); a nil end means iterate to
// the end of the keyspace.
type KVStore interface {
	Get(key []byte) []byte
	Set(key, value []byte)
	Delete(key []byte)
	Has(key []byte) bool
	Iterator(start, end []byte) Iterator
}

// Iterator mirrors cosmossdk.io/store/types.Iterator (subset).
type Iterator interface {
	Domain() (start, end []byte)
	Valid() bool
	Next()
	Key() []byte
	Value() []byte
	Close() error
}

// MemKVStore is an in-memory KVStore for tests and devnet bootstrap. It is
// not concurrency-safe — wrap with a mutex if the consumer touches it from
// multiple goroutines. Production keeper code runs inside the SDK BaseApp
// context which already serialises store access.
type MemKVStore struct {
	data map[string][]byte
}

// NewMemKVStore returns an empty in-memory store.
func NewMemKVStore() *MemKVStore {
	return &MemKVStore{data: make(map[string][]byte)}
}

func (m *MemKVStore) Get(key []byte) []byte {
	v, ok := m.data[string(key)]
	if !ok {
		return nil
	}
	out := make([]byte, len(v))
	copy(out, v)
	return out
}

func (m *MemKVStore) Set(key, value []byte) {
	stored := make([]byte, len(value))
	copy(stored, value)
	m.data[string(key)] = stored
}

func (m *MemKVStore) Delete(key []byte) {
	delete(m.data, string(key))
}

func (m *MemKVStore) Has(key []byte) bool {
	_, ok := m.data[string(key)]
	return ok
}

// Iterator returns a snapshot iterator over the half-open key range
// [start, end). A nil end means "to the end of keyspace".
func (m *MemKVStore) Iterator(start, end []byte) Iterator {
	keys := make([]string, 0, len(m.data))
	startStr := string(start)
	for k := range m.data {
		if start != nil && k < startStr {
			continue
		}
		if end != nil && k >= string(end) {
			continue
		}
		keys = append(keys, k)
	}
	// stable lexicographic order
	sortStrings(keys)
	return &memIterator{store: m, keys: keys, idx: 0, start: start, end: end}
}

type memIterator struct {
	store      *MemKVStore
	keys       []string
	idx        int
	start, end []byte
}

func (it *memIterator) Domain() (start, end []byte) { return it.start, it.end }
func (it *memIterator) Valid() bool                 { return it.idx < len(it.keys) }
func (it *memIterator) Next()                       { it.idx++ }
func (it *memIterator) Key() []byte                 { return []byte(it.keys[it.idx]) }
func (it *memIterator) Value() []byte               { return it.store.Get([]byte(it.keys[it.idx])) }
func (it *memIterator) Close() error                { return nil }

// sortStrings is a tiny insertion-sort helper to avoid pulling in sort just
// for the iterator. The map sizes we deal with in unit tests are tiny.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
