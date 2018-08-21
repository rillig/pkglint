* Of the user-defined variables, some may be used at load-time and some
  don't. Find out how pkglint can distinguish them.

* Make sure that no variable is modified at load-time after it has been
  used once. This should at least flag BUILD_DEFS in bsd.pkg.mk.

* ${MACHINE_ARCH}-${LOWER_OPSYS}elf in PLISTs etc. is a NetBSD config.guess
  problem ==> use of ${APPEND_ELF}

* If a dependency depends on an option (in options.mk), it should also
  depend on the same option in the buildlink3.mk file.

* don't complain about "procedure calls", like for pkg-build-options in
  the various buildlink3.mk files.

* if package A conflicts with B, then B should also conflict with A.

# Case-sensitive file systems

* Check for parallel files/dirs whose names differ only in case.

* When pkglint runs on a case-insensitive filesystem, it should still
  point out problems that only occur on case-sensitive filesystems. For
  example, devel/p5-Net-LDAP and devel/p5-Net-ldap should be considered
  different paths.

# Python

* Packages including lang/python/extension.mk must follow the Python version
  scheme. Enforcing PYPKGPREFIX for those is most likely a good idea.

* Warn about using REPLACE_PYTHON without including application.mk.

# Tools

```no-highlighting
# anfangs sind keine Tools definiert

read mk/tools/defaults.mk:
TOOLS_CREATE+= cat:CAT (+ weitere Eigenschaften?)
TOOLS_CREATE+= pax:PAX
TOOLS_CREATE+= grep:GREP_CMD

# cat ist ein pkgsrc-Tool mit Variable CAT
# pax ist ein pkgsrc-Tool mit Variable PAX
# grep ist ein pkgsrc-Tool mit Variable GREP_CMD

read mk/bsd.prefs.mk:
USE_TOOLS+= cat

# ${CAT} darf preproc-genutzt werden,
# sobald bsd.prefs.mk includiert wurde;
# pax und grep dürfen nicht

read mk/bsd.pkg.mk:
USE_TOOLS+= pax

# pax oder ${PAX} darf runtime-genutzt werden.

Makefile:

# pkgsrc-nutzbare Tools dürfen als ${VAR} runtime-genutzt werden
# pkgsrc-nutzbare Tools dürfen mit Namen in {pre,do,post}-* genutzt werden

USE_TOOLS += cat

# ${CAT} darf preproc-genutzt werden, nachdem bsd.prefs.mk includiert wurde

.include bsd.prefs.mk

# ab jetzt darf ${CAT} preproc-genutzt werden

USE_TOOLS += dog

# dog darf als ${DOG} runtime-genutzt werden
# dog darf als dog in {pre,do,post}-* genutzt werden

.include bsd.pkg.mk

# Schluss, Ende, aus
```
