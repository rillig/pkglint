package pkglint

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"io/ioutil"
	"path"
	"strings"
)

func CheckLinesDistinfo(lines Lines) {
	if trace.Tracing {
		defer trace.Call1(lines.FileName)()
	}

	filename := lines.FileName
	patchdir := "patches"
	if G.Pkg != nil && dirExists(G.Pkg.File(G.Pkg.Patchdir)) {
		patchdir = G.Pkg.Patchdir
	}
	if trace.Tracing {
		trace.Step1("patchdir=%q", patchdir)
	}

	distinfoIsCommitted := isCommitted(filename)
	ck := distinfoLinesChecker{
		lines, patchdir, distinfoIsCommitted,
		nil, make(map[string]distinfoFileInfo)}
	ck.parse()
	ck.check()
	CheckLinesTrailingEmptyLines(lines)
	ck.checkUnrecordedPatches()

	SaveAutofixChanges(lines)
}

type distinfoLinesChecker struct {
	lines               Lines
	patchdir            string // Relative to G.Pkg
	distinfoIsCommitted bool

	filenames []string // For keeping the order from top to bottom
	infos     map[string]distinfoFileInfo
}

func (ck *distinfoLinesChecker) parse() {
	lines := ck.lines

	llex := NewLinesLexer(lines)
	if lines.CheckRcsID(0, ``, "") {
		llex.Skip()
	}
	llex.SkipEmptyOrNote()

	prevFilename := ""
	var hashes []distinfoHash

	isPatch := func() YesNoUnknown {
		switch {
		case !hasPrefix(prevFilename, "patch-"):
			return no
		case G.Pkg == nil:
			return unknown
		case fileExists(G.Pkg.File(ck.patchdir + "/" + prevFilename)):
			return yes
		default:
			return no
		}
	}

	finishGroup := func() {
		ck.filenames = append(ck.filenames, prevFilename)
		ck.infos[prevFilename] = distinfoFileInfo{isPatch(), hashes}
		hashes = nil
	}

	for !llex.EOF() {
		line := llex.CurrentLine()
		llex.Skip()

		m, alg, filename, hash := match3(line.Text, `^(\w+) \((\w[^)]*)\) = (\S+)(?: bytes)?$`)
		if !m {
			line.Errorf("Invalid line: %s", line.Text)
			continue
		}

		if prevFilename != "" && filename != prevFilename {
			finishGroup()
		}
		prevFilename = filename

		hashes = append(hashes, distinfoHash{line, filename, alg, hash})
	}

	if prevFilename != "" {
		finishGroup()
	}
}

func (ck *distinfoLinesChecker) check() {
	for _, filename := range ck.filenames {
		info := ck.infos[filename]

		ck.checkAlgorithms(info)
		for _, hash := range info.hashes {
			ck.checkGlobalDistfileMismatch(hash)
			if info.isPatch == yes {
				ck.checkUncommittedPatch(hash)
			}
		}
	}
}

func (ck *distinfoLinesChecker) checkAlgorithms(info distinfoFileInfo) {
	filename := info.filename()
	algorithms := info.algorithms()
	line := info.line()

	isPatch := info.isPatch

	switch {
	case algorithms == "SHA1" && isPatch != no:
		return

	case algorithms == "SHA1, RMD160, SHA512, Size" && isPatch != yes:
		return
	}

	switch {
	case isPatch == yes:
		line.Errorf("Expected SHA1 hash for %s, got %s.", filename, algorithms)

	case isPatch == unknown:
		line.Errorf("Wrong checksum algorithms %s for %s.", algorithms, filename)
		line.Explain(
			"Distfiles that are downloaded from external sources must have the",
			"checksum algorithms SHA1, RMD160, SHA512, Size.",
			"",
			"Patch files from pkgsrc must have only the SHA1 hash.")

	// At this point, the file is either a missing patch file or a distfile.

	case hasPrefix(filename, "patch-") && algorithms == "SHA1":
		if G.Pkg.IgnoreMissingPatches {
			break
		}

		pathToPatchdir := relpath(path.Dir(line.Filename), G.Pkg.File(ck.patchdir))
		line.Warnf("Patch file %q does not exist in directory %q.", filename, pathToPatchdir)
		G.Explain(
			"If the patches directory looks correct, the patch may have been",
			"removed without updating the distinfo file.",
			"In such a case please update the distinfo file.",
			"",
			"In rare cases, pkglint cannot determine the correct location of the patches directory.",
			"In that case, see the pkglint man page for contact information.")

	default:
		line.Errorf("Expected SHA1, RMD160, SHA512, Size checksums for %q, got %s.", filename, algorithms)
	}
}

func (ck *distinfoLinesChecker) checkUnrecordedPatches() {
	if G.Pkg == nil {
		return
	}
	patchFiles, err := ioutil.ReadDir(G.Pkg.File(ck.patchdir))
	if err != nil {
		if trace.Tracing {
			trace.Stepf("Cannot read patchdir %q: %s", ck.patchdir, err)
		}
		return
	}

	for _, file := range patchFiles {
		patchName := file.Name()
		if file.Mode().IsRegular() && ck.infos[patchName].isPatch != yes && hasPrefix(patchName, "patch-") {
			ck.lines.Errorf("Patch %q is not recorded. Run %q.",
				cleanpath(relpath(path.Dir(ck.lines.FileName), G.Pkg.File(ck.patchdir+"/"+patchName))),
				bmake("makepatchsum"))
		}
	}
}

// Inter-package check for differing distfile checksums.
func (ck *distinfoLinesChecker) checkGlobalDistfileMismatch(info distinfoHash) {
	hashes := G.Hashes
	if hashes == nil {
		return
	}

	filename := info.filename
	alg := info.algorithm
	hash := info.hash
	line := info.line

	// Intentionally checking the filename instead of ck.isPatch.
	// Missing the few distfiles that actually start with patch-*
	// is more convenient than having lots of false positive mismatches.
	if hasPrefix(filename, "patch-") {
		return
	}

	// The Size hash is not encoded in hex, therefore it would trigger wrong error messages below.
	// Since the Size hash is targeted towards humans and not really useful for detecting duplicates,
	// omitting the check here is ok. Any mismatches will be reliably detected because the other
	// hashes will be different, too.
	if alg == "Size" {
		return
	}

	key := alg + ":" + filename
	otherHash := hashes[key]

	// See https://github.com/golang/go/issues/29802
	hashBytes := make([]byte, hex.DecodedLen(len(hash)))
	_, err := hex.Decode(hashBytes, []byte(hash))
	if err != nil {
		line.Errorf("The %s hash for %s contains a non-hex character.", alg, filename)
	}

	if otherHash != nil {
		if !bytes.Equal(otherHash.hash, hashBytes) {
			line.Errorf("The %s hash for %s is %s, which conflicts with %s in %s.",
				alg, filename, hash, hex.EncodeToString(otherHash.hash), line.RefToLocation(otherHash.location))
		}
	} else {
		hashes[key] = &Hash{hashBytes, line.Location}
	}
}

func (ck *distinfoLinesChecker) checkUncommittedPatch(info distinfoHash) {
	patchName := info.filename
	alg := info.algorithm
	hash := info.hash
	line := info.line

	patchFileName := ck.patchdir + "/" + patchName
	resolvedPatchFileName := G.Pkg.File(patchFileName)
	if ck.distinfoIsCommitted && !isCommitted(resolvedPatchFileName) {
		line.Warnf("%s is registered in distinfo but not added to CVS.", line.PathToFile(resolvedPatchFileName))
	}
	if alg == "SHA1" {
		ck.checkPatchSha1(line, patchFileName, hash)
	}
}

func (ck *distinfoLinesChecker) checkPatchSha1(line Line, patchFileName, distinfoSha1Hex string) {
	fileSha1Hex, err := computePatchSha1Hex(G.Pkg.File(patchFileName))
	if err != nil {
		line.Errorf("Patch %s does not exist.", patchFileName)
		return
	}
	if distinfoSha1Hex != fileSha1Hex {
		fix := line.Autofix()
		fix.Errorf("SHA1 hash of %s differs (distinfo has %s, patch file has %s).",
			line.PathToFile(G.Pkg.File(patchFileName)), distinfoSha1Hex, fileSha1Hex)
		fix.Explain(
			"To fix the hashes, either let pkglint --autofix do the work",
			sprintf("or run %q.", bmake("makepatchsum")))
		fix.Replace(distinfoSha1Hex, fileSha1Hex)
		fix.Apply()
	}
}

type distinfoFileInfo struct {
	//  yes     = the patch file exists
	//  unknown = distinfo file is checked without a pkgsrc package
	//  no      = distfile or nonexistent patch file
	isPatch YesNoUnknown
	hashes  []distinfoHash
}

func (info *distinfoFileInfo) filename() string { return info.hashes[0].filename }
func (info *distinfoFileInfo) line() Line       { return info.hashes[0].line }

func (info *distinfoFileInfo) algorithms() string {
	var algs []string
	for _, hash := range info.hashes {
		algs = append(algs, hash.algorithm)
	}
	return strings.Join(algs, ", ")
}

type distinfoHash struct {
	line      Line
	filename  string
	algorithm string
	hash      string
}

// Same as in mk/checksum/distinfo.awk:/function patchsum/
func computePatchSha1Hex(patchFilename string) (string, error) {
	patchBytes, err := ioutil.ReadFile(patchFilename)
	if err != nil {
		return "", err
	}

	hasher := sha1.New()
	skipText := []byte("$" + "NetBSD")
	for _, patchLine := range bytes.SplitAfter(patchBytes, []byte("\n")) {
		if !bytes.Contains(patchLine, skipText) {
			hasher.Write(patchLine)
		}
	}
	return sprintf("%x", hasher.Sum(nil)), nil
}
