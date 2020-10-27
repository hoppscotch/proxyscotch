package inputbox

import "github.com/martinlindhe/inputbox"

func InputBox(title, message, defaultInput string) (string, bool) {
	return inputbox.InputBox(title, message, defaultInput)
}
