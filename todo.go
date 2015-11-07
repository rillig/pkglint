package main

func notImplemented() {
	logError(NO_FILE, NO_LINES, "not implemented")
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
func checklineRcsid(line *Line, re, suggestedText string) {
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
func checkUnusedLicenses() {
	notImplemented()
}
func varIsDefined(varname string) bool {
	notImplemented()
	return false
}
func pkgverCmp(left, op, right string) bool {
	notImplemented()
	return false
}
func checklineMkShellcmdUse(line *Line, shellword string) {
	notImplemented()
}
func checklinesMk(lines []*Line) {
	notImplemented()
}
func varIsUsed(varname string) bool {
	notImplemented()
	return false
}
func checklineRelativePkgdir(line *Line, pkgdir string) {
	notImplemented()
}
func checklineMkVartypeBasic(line *Line, varname, basicType, op, value, comment string, listContext, guessed bool) {
	notImplemented()
}
func checklineMkShellword(line *Line, word string, _ bool) {
	notImplemented()
}
func checklineRelativePath(line *Line, path string, _ bool) {
	notImplemented()
}
