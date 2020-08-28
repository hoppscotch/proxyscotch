package inputbox

import (
	"github.com/gen2brain/dlgs"
)

func InputBox(title string, message string, defaultInput string)  (string, bool) {
	response, _, err := dlgs.Entry(title, message, defaultInput);
	return response, err == nil;
}
