package main

func notImplemented() {
	logErrorF(NO_FILE, NO_LINES, "not implemented")
}

func checkperms(fname string) {
	notImplemented()
}
func checklineLength(line *Line, maxchars int) {
	notImplemented()
}
func checklineTrailingWhitespace(line *Line) {
	notImplemented()
}
func checklineValidCharacters(line *Line, re string) {
	notImplemented()
}
func checklinesTrailingEmptyLines(lines []*Line) {
	notImplemented()
}
func parseLicenses(licensesSpec string) []string {
	notImplemented()
	return make([]string, 0)
}
func normalizePathname(fname string) string {
	notImplemented()
	return fname
}
func checklineMkAbsolutePathname(line *Line, text string) {
	notImplemented()
}
func checklineRcsid(line *Line, something string) {
	notImplemented()
}
func checkdirCvs(fname string) {
	notImplemented()
}
func checkfile(fname string) {
	notImplemented()
}
func checkdirPackage() {
	notImplemented()
}
func checkdirCategory() {
	notImplemented()
}
func checkdirToplevel() {
	notImplemented()
}
func checkUnusedLicenses() {
	notImplemented()
}
func expandVariableDef(varname string, defval string) *string {
	notImplemented()
	return &defval
}
func varIsDefined(varname string) bool {
	notImplemented()
	return false
}
func pkgverCmp(left, op, right string) bool {
	notImplemented()
	return false
}
func checklineMkVaruse(line *Line, varname, mod string, vuctx *VarUseContext) {
	notImplemented()
}
func checklineMkShellcmdUse(line *Line, shellword string) {
	notImplemented()
}
