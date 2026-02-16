module github.com/freeformz/sets

go 1.25

toolchain go1.26.0

retract (
    // breaking change that was reverted in 0.10.0
    v0.9.0
    v0.9.1 
)

require github.com/google/go-cmp v0.7.0

require pgregory.net/rapid v1.2.0
