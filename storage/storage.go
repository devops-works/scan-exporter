package storage

// Store is a key/value
type Store map[string][]string

// Add a value to the store
func (s Store) Add(k, v string) {
	s[k] = append(s[k], v)
}

// Create initalize the store
func Create() Store {
	s := make(Store)
	return s
}

// Delete a key from the store
func (s Store) Delete(k string) {
	delete(s, k)
}

// Get the values associated to a key from the store
func (s Store) Get(k string) []string {
	return s[k]
}

// Update a value in the store
func (s Store) Update(k string, v []string) {
	s[k] = v
}
