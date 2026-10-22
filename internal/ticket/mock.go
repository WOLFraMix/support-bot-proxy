package ticket

import (
	"fmt"
	"math/rand/v2"
)

func GenerateTicketKey() string {
	return fmt.Sprintf("TEST-%d", rand.IntN(100000))
}
