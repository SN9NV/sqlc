package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/trace"
	"sort"
	"strings"

	"github.com/cubicdaiya/gonp"
	"github.com/kyleconroy/sqlc/internal/debug"
)

func Diff(ctx context.Context, e Env, dir, name string, stderr io.Writer) error {
	output, err := Generate(ctx, e, dir, name, stderr)
	if err != nil {
		return err
	}
	if debug.Traced {
		defer trace.StartRegion(ctx, "checkfiles").End()
	}
	var errored bool

	keys := make([]string, 0, len(output))
	for k, _ := range output {
		kk := k
		keys = append(keys, kk)
	}
	sort.Strings(keys)

	for _, filename := range keys {
		source := output[filename]
		if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
			errored = true
			// stdout message
			continue
		}
		existing, err := os.ReadFile(filename)
		if err != nil {
			errored = true
			fmt.Fprintf(stderr, "%s: %s\n", filename, err)
			continue
		}
		diff := gonp.New(getLines(existing), getLines([]byte(source)))
		diff.Compose()
		uniHunks := filterHunks(diff.UnifiedHunks())

		if len(uniHunks) > 0 {
			errored = true
			fmt.Fprintf(stderr, "--- a%s\n", strings.TrimPrefix(filename, dir))
			fmt.Fprintf(stderr, "+++ b%s\n", strings.TrimPrefix(filename, dir))
			diff.FprintUniHunks(stderr, uniHunks)
		}
	}
	if errored {
		return errors.New("diff found")
	}
	return nil
}
