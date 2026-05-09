package narr

import "embed"

// BundledSkills contains the repository skills packaged into the narrc binary.
//
//go:embed skills/*
var BundledSkills embed.FS
