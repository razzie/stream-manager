package template

import _ "embed"

//go:embed streams.html
var Streams string

//go:embed launch.html
var Launch string

//go:embed probe.html
var Probe string
