package main

import (
	"bytes"
	"crypto/sha1"
	"io/ioutil"
	"strings"
)

type DistinfoChecker struct {
	previousFilename string
	isPatch          bool
	seenAlgorithms   []string
}

func checkfileDistinfo(fname string) {
	defer tracecall("checkfileDistinfo", fname)()

	lines := loadNonemptyLines(fname, false)
	if lines == nil {
		return
	}

	distinfoIsCommitted := isCommitted(fname)

	checklineRcsid(lines[0], ``, "")
	if 1 < len(lines) && lines[1].text != "" {
		lines[1].notef("Empty line expected.")
	}

	patchesDir := G.pkgContext.patchdir
	if patchesDir == "" && dirExists(G.currentDir+"/patches") {
		patchesDir = "patches"
	}

	ctx := &DistinfoChecker{"", false, make([]string, 0)}
	inDistinfo := make(map[string]bool)
	for i, line := range lines {
		if i < 2 {
			continue
		}
		m, alg, fname, hash := match3(line.text, `^(\w+) \(([^)]+)\) = (.*)(?: bytes)?$`)
		if !m {
			line.errorf("Unknown line type.")
			continue
		}

		if fname != ctx.previousFilename {
			ctx.onFilenameChange(line, fname)
		}

		if !matches(fname, `^\w`) {
			line.errorf("All file names must start with a letter.")
		}

		// Inter-package check for differing distfile checksums.
		if G.ipcDistinfo != nil && !ctx.isPatch {
			key := alg + ":" + fname
			otherHash := G.ipcDistinfo[key]
			if otherHash != nil {
				if otherHash.hash != hash {
					line.errorf("The hash %s for %s is %s, ...", alg, fname, hash)
					otherHash.line.errorf("... which differs from %s.", otherHash.hash)
				}
			} else {
				G.ipcDistinfo[key] = &Hash{hash, line}
			}
		}

		ctx.seenAlgorithms = append(ctx.seenAlgorithms, alg)

		if ctx.isPatch && G.pkgContext.distinfoFile != "./../../lang/php5/distinfo" {
			fname := G.currentDir + "/" + patchesDir + "/" + fname
			if distinfoIsCommitted && !isCommitted(fname) {
				line.warnf("%s/%s is registered in distinfo but not added to CVS.", patchesDir, fname)
			}
			ctx.checkPatchSha1(line, fname, hash)
		}
		inDistinfo[fname] = true

	}
	ctx.onFilenameChange(NewLine(fname, NO_LINES, "", nil), "")
	checklinesTrailingEmptyLines(lines)

	files, err := ioutil.ReadDir(G.currentDir + "/" + patchesDir)
	if err != nil {
		for _, file := range files {
			patch := file.Name()
			if !inDistinfo[patch] {
				errorf(fname, NO_LINES, "patch is not recorded. Run \"%s makepatchsum\".", confMake)
			}
		}
	}
}

func (ctx *DistinfoChecker) onFilenameChange(line *Line, fname string) {
	if ctx.previousFilename != "" {
		hashAlgorithms := strings.Join(ctx.seenAlgorithms, ", ")
		_ = G.opts.optDebugMisc && line.debugf("File %s is hashed with %v.", fname, ctx.seenAlgorithms)
		if ctx.isPatch {
			if hashAlgorithms != "SHA1" {
				line.errorf("Expected SHA1 hash for %s, got %s.", fname, hashAlgorithms)
			}
		} else {
			if hashAlgorithms != "SHA1, RMD160, Size" && hashAlgorithms != "SHA1, RMD160, SHA512, Size" {
				line.errorf("Expected SHA1, RMD160, SHA512, Size checksums for %s, got %s.", fname, hashAlgorithms)
			}
		}
	}

	ctx.isPatch = matches(fname, `^patch-.+$`)
	ctx.previousFilename = fname
	ctx.seenAlgorithms = make([]string, 0)
}

func (ctx *DistinfoChecker) checkPatchSha1(line *Line, fname, distinfoSha1Hex string) {
	patchBytes, err := ioutil.ReadFile(fname)
	if err != nil {
		line.errorf("%s does not exist.", fname)
		return
	}

	h := sha1.New()
	netbsd := []byte("$NetBSD")
	for _, patchLine := range bytes.SplitAfter(patchBytes, []byte("\n")) {
		if !bytes.Contains(patchLine, netbsd) {
			h.Write(patchLine)
		}
	}
	fileSha1Hex := sprintf("%x", h.Sum(nil))
	if distinfoSha1Hex != fileSha1Hex {
		line.errorf("%s hash of %s differs (distinfo has %s, patch file has %s). Run \"%s makepatchsum\".", "SHA1", fname, distinfoSha1Hex, fileSha1Hex, confMake)
	}
}
