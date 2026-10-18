module example.test/solution

go 1.27.0

toolchain go1.27.2

require (
 example.org/catalogue v1.2.0
 example.org/library v0.4.0 // indirect
)

exclude example.org/catalogue v1.1.0

replace (
 example.org/catalogue v1.2.0 => example.net/catalogue v1.2.1
 example.org/library => "../local library"
)
