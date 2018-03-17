package main

// Pkgsrc describes a pkgsrc installation.
// In each pkglint run, only a single pkgsrc installation is ever loaded.
// It just doesn't make sense to check multiple pkgsrc installations at once.
type Pkgsrc = *PkgsrcImpl

type PkgsrcImpl struct {

	// The top directory (PKGSRCDIR), either absolute or relative to
	// the current working directory.
	topdir string
}

func NewPkgsrc(dir string) Pkgsrc {
	return &PkgsrcImpl{dir}
}

// LoadExistingLines loads the file relative to the pkgsrc top directory.
func (src *PkgsrcImpl) LoadExistingLines(fileName string, joinBackslashLines bool) []Line {
	return LoadExistingLines(src.topdir+"/"+fileName, joinBackslashLines)
}

// File resolves a file name relative to the pkgsrc top directory.
//
// Example:
//  NewPkgsrc("/usr/pkgsrc").File("distfiles") => "/usr/pkgsrc/distfiles"
func (src *PkgsrcImpl) File(relativeName string) string {
	return src.topdir + "/" + relativeName
}

// ToRel returns the path of `fileName`, relative to the pkgsrc top directory.
//
// Example:
//  NewPkgsrc("/usr/pkgsrc").ToRel("/usr/pkgsrc/distfiles") => "distfiles"
func (src *PkgsrcImpl) ToRel(fileName string) string {
	return relpath(src.topdir, fileName)
}
