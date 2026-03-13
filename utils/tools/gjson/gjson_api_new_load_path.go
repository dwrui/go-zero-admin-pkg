package gjson

import "github.com/dwrui/go-zero-admin/pkg/utils/tools/gfile"

// Load loads content from specified file `path`, and creates a Json object from its content.
// Deprecated: use LoadPath instead.
func Load(path string, safe ...bool) (*Json, error) {
	var isSafe bool
	if len(safe) > 0 {
		isSafe = safe[0]
	}
	return LoadPath(path, Options{
		Safe: isSafe,
	})
}

// LoadPath loads content from specified file `path`, and creates a Json object from its content.
func LoadPath(path string, options Options) (*Json, error) {
	if p, err := gfile.Search(path); err != nil {
		return nil, err
	} else {
		path = p
	}
	if options.Type == "" {
		options.Type = ContentType(gfile.Ext(path))
	}
	return loadContentWithOptions(gfile.GetBytesWithCache(path), options)
}
