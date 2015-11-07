package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"strings"
)

type DistinfoContext struct {
	previousFilename string
	isPatch          bool
	seenAlgorithms   []string
}

func checkfileDistinfo(fname string) {
	//my (lines, %in_distinfo, patches_dir, di_is_committed

	_ = G.opts.optDebugTrace && logDebug(fname, NO_LINES, "checkfileDistinfo()")

	lines := loadNonemptyLines(fname, false)
	if lines == nil {
		return
	}

	distinfoIsCommitted := isCommitted(fname)

	checklineRcsid(lines[0], ``, "")
	if 1 < len(lines) && lines[1].text != "" {
		lines[1].logNote("Empty line expected.")
	}

	patchesDir := G.pkgContext.patchdir
	if patchesDir == "" && dirExists(G.currentDir+"/patches") {
		patchesDir = "patches"
	}

	ctx := &DistinfoContext{"", false, make([]string, 0)}
	inDistinfo := make(map[string]bool)
	for _, line := range lines[2:] {
		m, alg, fname, hash := match3(line.text, `^(\w+) \(([^)]+)\) = (.*)(?: bytes)?$`)
		if !m {
			line.logError("Unknown line type.")
			continue
		}

		if fname != ctx.previousFilename {
			ctx.onFilenameChange(line, fname)
		}

		if !match0(fname, `^\w`) {
			line.logError("All file names must start with a letter.")
		}

		// Inter-package check for differing distfile checksums.
		if G.opts.optCheckGlobal && !ctx.isPatch {
			otherHash := G.ipcDistinfo[alg+":"+fname]
			if otherHash != nil {
				if otherHash.hash != hash {
					line.logError("The hash %s for %s is %s, ...", alg, fname, hash)
					otherHash.line.logError("... which differs from %s.", otherHash.hash)
				}
			} else {
				G.ipcDistinfo[alg+":"+fname] = &Hash{hash, line}
			}
		}

		ctx.seenAlgorithms = append(ctx.seenAlgorithms, alg)

		if ctx.isPatch && G.pkgContext.distinfoFile != "./../../lang/php5/distinfo" {
			fname := G.currentDir + "/" + patchesDir + "/" + fname
			if distinfoIsCommitted && !isCommitted(fname) {
				line.logWarning("${patches_dir}/${chksum_fname} is registered in distinfo but not added to CVS.")
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
				logError(fname, NO_LINES, "patch is not recorded. Run \"%s makepatchsum\".", confMake)
			}
		}
	}
}

func (ctx *DistinfoContext) onFilenameChange(line *Line, fname string) {
	if ctx.previousFilename != "" {
		hashAlgorithms := strings.Join(ctx.seenAlgorithms, ", ")
		_ = G.opts.optDebugMisc && line.logDebug("File %s is hashed with %v.", fname, ctx.seenAlgorithms)
		if ctx.isPatch {
			if hashAlgorithms != "SHA1" {
				line.logError("Expected SHA1 hash for %s, got %s.", fname, hashAlgorithms)
			}
		} else {
			if hashAlgorithms != "SHA1, RMD160, Size" && hashAlgorithms != "SHA1, RMD160, SHA512, Size" {
				line.logError("Expected SHA1, RMD160, SHA512, Size checksums for %s, got %s.", fname, hashAlgorithms)
			}
		}
	}

	ctx.isPatch = match0(fname, `^patch-.+$`)
	ctx.previousFilename = fname
	ctx.seenAlgorithms = make([]string, 0)
}

func (ctx *DistinfoContext) checkPatchSha1(line *Line, fname, distinfoSha1Hex string) {
	patchBytes, err := ioutil.ReadFile(fname)
	if err != nil {
		line.logError("%s does not exist.", fname)
		return
	}

	h := sha1.New()
	netbsd := []byte("$NetBSD")
	for _, patchLine := range bytes.SplitAfter(patchBytes, []byte("\n")) {
		if !bytes.Contains(patchLine, netbsd) {
			h.Write(patchLine)
		}
	}
	fileSha1Hex := fmt.Sprintf("%x", h.Sum(nil))
	if distinfoSha1Hex != fileSha1Hex {
		line.logError("%s hash of %s differs (distinfo has %s, patch file has %s). Run \"%s makepatchsum\".", "SHA1", fname, distinfoSha1Hex, fileSha1Hex, confMake)
	}
}
