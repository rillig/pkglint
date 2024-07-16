# 23.5.0 (2024-07-16)

Added the -Werror option, which treats warnings as errors.

# 23.4.2 (2024-05-07)

Fixed the wrong suggestion that list variables or variables of an unknown type
could be compared using `${LIST} == word` instead of `${LIST:Mword}`.

Fixed the wrong warning when a value is appended using `+=` to a variable of
unknown type.
