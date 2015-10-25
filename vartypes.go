package main

func parseAclEntries(args []string) []AclEntry {
	result := make([]AclEntry, 0)
	for _, arg := range args {
		m := mustMatch(`^([\w.*]+|_):([adpsu]*)$`, arg)
		glob, perms := m[1], m[2]
		result = append(result, AclEntry{glob, perms})
	}
	return result
}
func loadVartypesBasictypes() {
	panic("not implemented")
}
