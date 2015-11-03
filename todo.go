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
func checklineMkAbsolutePathname(line *Line, text string) {
	line.logError("not implemented")
}
func checklineRcsid(line *Line, something string) {
	line.logErrorF("not implemented")
}

func checkdirCvs(fname string) {
	panic("not implemented")
}
func checkfile(fname string) {
	panic("not implemented")
}
func checkdirPackage() {
	panic("not implemented")
}
func checkdirCategory() {
	panic("not implemented")
}
func checkdirToplevel() {
	panic("not implemented")
}

func checkUnusedLicenses() {
	panic("not implemented")
}
func expandVariableDef(varname string, defval string) *string {
	panic("not implemented")
	return &defval
}
func determineUsedVariables(lines []*Line) {
	panic("not implemented")
}
func varIsDefined(varname string) bool {
	panic("not implemented")
	return false
}
