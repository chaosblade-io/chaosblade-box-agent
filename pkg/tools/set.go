package tools

type Set struct {
	m map[interface{}]bool
}

func NewSet() *Set {
	return &Set{
		m: make(map[interface{}]bool),
	}
}

func (set *Set) Add(item interface{}) {
	set.m[item] = true
}

func (set *Set) Remove(item interface{}) {
	delete(set.m, item)
}

func (set *Set) Contains(item interface{}) bool {
	ok := set.m[item]
	return ok
}

func (set *Set) Length() int {
	return len(set.m)
}

func (set *Set) Clear() {
	set.m = map[interface{}]bool{}
}

func (set *Set) Keys() []interface{} {
	keys := make([]interface{}, 0, len(set.m))
	for key := range set.m {
		keys = append(keys, key)
	}
	return keys
}

func (set *Set) StringKeys() []string {
	keys := make([]string, 0, len(set.m))
	for key := range set.m {
		keys = append(keys, key.(string))
	}
	return keys
}
