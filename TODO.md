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

# Misc

```
do-install:
	${ECHO} ${msg}   # Should produce: Undefined variable ${msg}.
.for msg in message1
	${ECHO} ${msg}
.endfor
```

* Check all diagnostics that refer to another file.
  The path to that file must be given relative to the diagnostic line.

* Check all warnings and errors whether their explanation has instructions
  on how to fix the diagnostic properly.

* Ensure even better test coverage than 100%.
  For each of the testees, there should be 100% code coverage by
  only those tests whose name corresponds to the testee.
