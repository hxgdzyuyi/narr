package state

import (
	"sort"

	"narr/internal/model"
)

type FieldKey struct {
	Entity model.SymbolID
	Field  string
}

func (k FieldKey) String() string {
	return string(k.Entity) + "." + k.Field
}

type Store struct {
	values map[FieldKey]Value
}

func NewStore() Store {
	return Store{values: map[FieldKey]Value{}}
}

func (s Store) Clone() Store {
	copy := NewStore()
	for key, value := range s.values {
		copy.values[key] = value
	}
	return copy
}

func (s Store) Get(key FieldKey) Value {
	value, ok := s.values[key]
	if !ok {
		return Missing()
	}
	return value
}

func (s Store) Set(key FieldKey, value Value) {
	s.values[key] = value
}

func (s Store) AddToSet(key FieldKey, value Value) {
	current := s.Get(key)
	if current.Kind != ValueSet {
		current = Set(nil)
	}
	current.Items = append(current.Items, value)
	s.values[key] = Set(current.Items)
}

func (s Store) RemoveFromSet(key FieldKey, value Value) {
	current := s.Get(key)
	if current.Kind != ValueSet {
		return
	}
	removeKey := value.StableKey()
	items := make([]Value, 0, len(current.Items))
	for _, item := range current.Items {
		if item.StableKey() != removeKey {
			items = append(items, item)
		}
	}
	s.values[key] = Set(items)
}

func (s Store) AppendToList(key FieldKey, value Value) {
	current := s.Get(key)
	if current.Kind != ValueList {
		current = List(nil)
	}
	items := append(append([]Value(nil), current.Items...), value)
	s.values[key] = List(items)
}

func (s Store) Keys() []FieldKey {
	keys := make([]FieldKey, 0, len(s.values))
	for key := range s.values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})
	return keys
}

func (s Store) Len() int {
	return len(s.values)
}
