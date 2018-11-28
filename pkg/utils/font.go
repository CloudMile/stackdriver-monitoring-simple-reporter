package utils

import (
	"github.com/golang/freetype/truetype"
	"io/ioutil"
	"sync"
)

var (
	_defaultFontLock sync.Mutex
	_defaultFont     *truetype.Font
)

func readFont() []byte {
	ttfBytes, err := ioutil.ReadFile("font/Hack-Regular.ttf")
	if err != nil {
		return nil
	}

	return ttfBytes
}

func GetFont() *truetype.Font {
	if _defaultFont == nil {
		_defaultFontLock.Lock()
		defer _defaultFontLock.Unlock()
		if _defaultFont == nil {
			font, err := truetype.Parse(readFont())
			if err != nil {
				return nil
			}
			_defaultFont = font
		}
	}
	return _defaultFont
}
