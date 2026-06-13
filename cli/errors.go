package cli

import (
	"errors"

	"github.com/tamnd/acmqueue-cli/acmqueue"
)

func isNotFound(err error) bool {
	return errors.Is(err, acmqueue.ErrNotFound)
}
