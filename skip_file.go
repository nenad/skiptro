package skiptro

import (
	"bytes"
	"fmt"
	"path"
	"time"
)

func EDL(scene Scene) []byte {
	return []byte(fmt.Sprintf("%.2f %.2f 3\n", scene.Start.Seconds(), scene.End.Seconds()))
}

func M3U(scene Scene, filename string) []byte {
	start := int(scene.Start.Seconds())
	end := int(scene.End.Seconds())
	name := path.Base(filename)
	b := bytes.Buffer{}

	preIntro := ""
	if scene.Start.Truncate(time.Second) > 0 {
		preIntro = fmt.Sprintf(`
#EXTVLCOPT:start-time=0
#EXTVLCOPT:stop-time=%d
#EXTINF ,PreIntro
%s
`, start, name)
	}

	b.WriteString(fmt.Sprintf(`#EXTM3U
#EXTM3U%s
#EXTVLCOPT:start-time=%d
#EXTINF ,PostIntro
%s
`,
		preIntro,
		end,
		name))

	return b.Bytes()
}
