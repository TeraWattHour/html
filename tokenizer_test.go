package html

import (
	"fmt"
	"testing"
)

func TestTokenize(t *testing.T) {
	template := `<!DOCTYPE html SYSTEM 'about:legacy-compat'><div id="con" data-count='data1-23' a13="abc" aaa="" data-13='true'> 5 < 5 </div>`

	for token := range Tokenize(template) {
		switch token := token.(type) {
		case *Illegal:
			t.Error(token)
			return
		case *StartTag:
			fmt.Println(token)
		case *Text:
			fmt.Println(token)
		case *EndTag:
			fmt.Println(token)
		}
	}
}
