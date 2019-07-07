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

* Check all warnings and errors whether their explanation has instructions
  on how to fix the diagnostic properly.

* Ensure even better test coverage than 100%.
  For each of the testees, there should be 100% code coverage by
  only those tests whose name corresponds to the testee.

* Implement the alignment rule for continuation backslashes in column 72,
  especially when autofixing the indentation.

### Test_VaralignBlock__tabbed_outlier
 
~~~
.if !empty(PKG_OPTIONS:Minspircd-sqloper)
INSPIRCD_STORAGE_DRIVER?=	mysql
MODULES+=		m_sqloper.cpp m_sqlutils.cpp
HEADERS+=		m_sqlutils.h
.endif

2: Breite 26, eingerückt mit Tab auf 33
3: Breite 9, eingerückt mit Tabs auf 25
4: Breite 9, eingerückt mit Tabs auf 25

unschön?
Ja, die Einrückung ist nicht einheitlich: 2x25, 1x33.

Möglichkeit 1: die 2x25 auf 33 erhöhen.

* Die Einrückung ist dann einheitlich.
* Die maximale Zeilenlänge wäre dann 53 + 8 = 61.
* Das liegt unterhalb von 72, daher ist es akzeptabel.

.if !empty(PKG_OPTIONS:Minspircd-sqloper)
INSPIRCD_STORAGE_DRIVER?=	mysql
MODULES+=			m_sqloper.cpp m_sqlutils.cpp
HEADERS+=			m_sqlutils.h
.endif

Möglichkeit 2: ist Zeile 2 ein Ausreißer?

* Es gibt keine Fortsetzungszeilen, das macht die Sache einfach.
* Zeile 2 hat die Einrückung 33, das ist 8 mehr als die zweitmeiste.
* Das reicht nicht für einen Ausreißer.
* Die übrigen Zeilen im Absatz sind konsistent eingerückt.
* Die übrigen Zeilen im Absatz sind weiter eingerückt als eigentlich nötig.
* Nach dem Entfernen der unnötigen Einrückung ist die zweittiefste Einrückung noch 17.
* Der Unterschied zwischen der 17 (korrigiert) und der 26 (mindest) reicht für einen Ausreißer.
* Zeile 2 ist nach der Umformung ein Ausreißer.
* Zeile 2 wird mit Leerzeichen statt Tab eingerückt.
* Zeilen 3 und 4 werden minimal eingerückt, also auf die 17.

.if !empty(PKG_OPTIONS:Minspircd-sqloper)
INSPIRCD_STORAGE_DRIVER?= mysql
MODULES+=	m_sqloper.cpp m_sqlutils.cpp
HEADERS+=	m_sqlutils.h
.endif
~~~
