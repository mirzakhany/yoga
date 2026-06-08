package render

import _ "embed"

// SourceSans3TTF is the embedded UI sans-serif face (OFL).
//
//go:embed assets/SourceSans3-Regular.ttf
var SourceSans3TTF []byte

// SourceCodeProTTF is the embedded monospace face for the code editor (OFL).
//
//go:embed assets/SourceCodePro-Regular.ttf
var SourceCodeProTTF []byte
