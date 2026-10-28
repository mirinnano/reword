package content

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Load reads every content file under root. Layout:
//
//	<grade>/vocab/*.yaml
//	<grade>/questions/<type>/*.yaml
//
// Files are parsed strictly (unknown fields are errors) and the grade/type
// declared inside a file must match its directory.
func Load(root string) (*Set, error) {
	set := &Set{Vocab: map[string][]Located[VocabItem]{}, Questions: map[string][]Located[Question]{}}
	var errs []error
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !(strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		parts := strings.Split(filepath.ToSlash(rel), "/")
		switch {
		case len(parts) == 3 && parts[1] == "vocab":
			var f VocabFile
			if err := decodeStrict(path, &f); err != nil {
				errs = append(errs, err)
				return nil
			}
			if f.Grade != parts[0] {
				errs = append(errs, fmt.Errorf("%s: grade %q does not match directory %q", rel, f.Grade, parts[0]))
				return nil
			}
			for _, it := range f.Items {
				set.Vocab[f.Grade] = append(set.Vocab[f.Grade], Located[VocabItem]{File: rel, Item: it})
			}
		case len(parts) == 4 && parts[1] == "questions":
			var f QuestionFile
			if err := decodeStrict(path, &f); err != nil {
				errs = append(errs, err)
				return nil
			}
			if f.Grade != parts[0] || f.Type != parts[2] {
				errs = append(errs, fmt.Errorf("%s: grade/type %q/%q does not match directory %q/%q", rel, f.Grade, f.Type, parts[0], parts[2]))
				return nil
			}
			for _, it := range f.Items {
				set.Questions[f.Type] = append(set.Questions[f.Type], Located[Question]{File: rel, Item: it})
			}
		default:
			errs = append(errs, fmt.Errorf("%s: unexpected location (see docs/content-specs/README.md)", rel))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return set, errors.Join(errs...)
}

func decodeStrict(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
