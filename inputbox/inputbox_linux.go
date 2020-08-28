package inputbox

import "github.com/martinlindhe/inputbox"

func InputBox(title string, message string, defaultInput string)  (string, bool) {
	return inputbox.InputBox(title, message, defaultInput);
}
