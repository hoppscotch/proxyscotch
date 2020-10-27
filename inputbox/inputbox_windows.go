package inputbox

import (
	"github.com/gen2brain/dlgs"
)

func InputBox(title, message, defaultInput string) (string, bool) {
	response, _, err := dlgs.Entry(title, message, defaultInput)
	return response, err == nil
}
