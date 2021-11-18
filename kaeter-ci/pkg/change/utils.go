package change

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func remove(a []string, i int) []string {
	a[i] = a[len(a)-1]
	return a[:len(a)-1]
}
