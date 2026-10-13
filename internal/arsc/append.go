package arsc

// Append adds a new string to the pool and returns its index.
// If the string already exists, returns the existing index.
func (p *StringPool) Append(s string) uint32 {
	for i, existing := range p.Strings {
		if existing == s {
			return uint32(i)
		}
	}
	p.Strings = append(p.Strings, s)
	return uint32(len(p.Strings) - 1)
}

// AppendString adds a new string to the global pool of the table.
func (t *Table) AppendString(s string) uint32 {
	if t.GlobalPool == nil {
		t.GlobalPool = &StringPool{IsUTF8: true}
	}
	return t.GlobalPool.Append(s)
}
