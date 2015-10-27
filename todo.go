package main

func checkperms(fname string) {
	logError(NO_FILE, NO_LINES, "not implemented")
}
func checklineLength(line *Line, maxchars int) {
	line.logError("not implemented")
}
func checklineTrailingWhitespace(line *Line) {
	line.logError("not implemented")
}
func checklineValidCharacters(line *Line, re string) {
	line.logError("not implemented")
}
func checklinesTrailingEmptyLines(lines []*Line) {
	logError(NO_FILE, NO_LINES, "not implemented")
}
func getVariableType(line *Line, varname string) *Type {
	logError(NO_FILE, NO_LINES, "not implemented")
	return nil
}
func parseLicenses(licensesSpec string) []string {
	logError(NO_FILE, NO_LINES, "not implemented")
	return make([]string, 0)
}
func normalizePathname(fname string) string {
	logError(NO_FILE, NO_LINES, "not implemented")
	return fname
}
