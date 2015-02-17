package objloader

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Vec4 struct {
	X float32
	Y float32
	Z float32
	W float32
}

type Vec3 struct {
	X float32
	Y float32
	Z float32
}

type Vertex struct {
	Position Vec4
	Normal   Vec3
	TexCoord Vec3
}

type Face struct {
	Vertices []Vertex
}

type errScanner struct {
	scanner *bufio.Scanner
	err     error
}

type numParser struct {
	err error
}

func (s *errScanner) scan() string {
	if s.err != nil {
		return ""
	}
	defer func() {
		if r := recover(); r != nil || s.scanner.Err() != nil {
			if r != nil {
				s.err = errors.New("Scanner Error")
			} else {
				s.err = s.scanner.Err()
			}
		}
	}()
	s.scanner.Scan()
	return s.scanner.Text()
}

func (s *errScanner) advance() {
	if s.err != nil {
		return
	}
	s.scanner.Scan()
}

func (fp *numParser) parseFloat(text string) float32 {
	if fp.err != nil {
		return 0
	}
	f, err := strconv.ParseFloat(text, 32)
	fp.err = err
	return float32(f)
}

func (ip *numParser) parseInt(text string) int64 {
	if ip.err != nil {
		return 0
	}
	i, err := strconv.ParseInt(text, 10, 0)
	ip.err = err
	return i
}

func checkErr(s errScanner, fp numParser, msg string) {
	if s.err != nil || fp.err != nil {
		panic(msg)
	}
}

func parseVertex(line string, positions *[]Vec4,
	normals *[]Vec3, texcoords *[]Vec3) {
	errText := "Invalid vertex format: " + line
	s := bufio.NewScanner(strings.NewReader(line))
	s.Split(bufio.ScanWords)
	es := &errScanner{err: nil, scanner: s}
	fp := numParser{}
	switch es.scan() {
	case "v":
		x := fp.parseFloat(es.scan())
		y := fp.parseFloat(es.scan())
		z := fp.parseFloat(es.scan())
		var w float32
		if es.scanner.Scan() {
			w = fp.parseFloat(es.scanner.Text())
		} else {
			w = 1.0
		}
		checkErr(*es, fp, errText)

		v := Vec4{X: x, Y: y, Z: z, W: w}
		*positions = append(*positions, v)
	case "vt":
		u := fp.parseFloat(es.scan())
		v := fp.parseFloat(es.scan())
		var w float32
		if es.scanner.Scan() {
			w = fp.parseFloat(es.scanner.Text())
		} else {
			w = 0
		}
		checkErr(*es, fp, errText)

		vt := Vec3{X: u, Y: v, Z: w}
		*texcoords = append(*texcoords, vt)
	case "vn":
		x := fp.parseFloat(es.scan())
		y := fp.parseFloat(es.scan())
		z := fp.parseFloat(es.scan())
		checkErr(*es, fp, errText)

		vn := Vec3{X: x, Y: y, Z: z}
		*normals = append(*normals, vn)
	case "vp":
		// Nothing!
	default:
		panic(errText)
	}
}

func splitSlash(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i := 0; i < len(data); i++ {
		if data[i] == '/' {
			return i + 1, data[:i], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func fixNdx(ndx int64, maxLen int) (fndx int64) {
	if ndx > 0 {
		fndx = ndx - 1
	} else if ndx < 0 {
		fndx = ndx + int64(maxLen)
	} else {
		fndx = 0
	}
	return
}

func getVec3(vecs *[]Vec3, ndx int64) Vec3 {
	return (*vecs)[fixNdx(ndx, len(*vecs))]
}

func getVec4(vecs *[]Vec4, ndx int64) Vec4 {
	return (*vecs)[fixNdx(ndx, len(*vecs))]
}

func parseFace(line string, positions *[]Vec4,
	normals *[]Vec3, texcoords *[]Vec3) Face {
	errText := "Invalid face format: " + line
	s := bufio.NewScanner(strings.NewReader(line))
	s.Split(bufio.ScanWords)
	if !s.Scan() || s.Text() != "f" {
		panic(errText)
	}

	face := Face{Vertices: make([]Vertex, 0, 3)}

	for s.Scan() {
		data := s.Text()
		var vert Vertex
		s := bufio.NewScanner(strings.NewReader(data))
		s.Split(splitSlash)
		es := errScanner{scanner: s}
		ip := numParser{}
		switch strings.Count(data, "/") {
		case 0:
			ndx, err := strconv.ParseInt(data, 10, 0)
			if err != nil {
				panic(errText)
			}
			vert.Position = getVec4(positions, ndx)
		case 1:
			ndx1 := ip.parseInt(es.scan())
			ndx2 := ip.parseInt(es.scan())
			checkErr(es, ip, errText)
			vert.Position = getVec4(positions, ndx1)
			vert.TexCoord = getVec3(texcoords, ndx2)
		case 2:
			ndx1 := ip.parseInt(es.scan())
			if 1 == strings.Count(data, "//") {
				es.advance()
				ndx2 := ip.parseInt(es.scan())
				checkErr(es, ip, errText)
				vert.Position = getVec4(positions, ndx1)
				vert.Normal = getVec3(normals, ndx2)
			} else {
				ndx2 := ip.parseInt(es.scan())
				ndx3 := ip.parseInt(es.scan())
				checkErr(es, ip, errText)
				vert.Position = getVec4(positions, ndx1)
				vert.TexCoord = getVec3(texcoords, ndx2)
				vert.Normal = getVec3(normals, ndx3)
			}
		default:
			panic(errText)
		}
		face.Vertices = append(face.Vertices, vert)
	}

	return face
}

func parseGroup(line string) {

}

func parseObject(line string) {

}

func LoadObj(filename string) []Face {
	faces := make([]Face, 0, 20)
	positions := make([]Vec4, 0, 20)
	normals := make([]Vec3, 0, 20)
	texcoords := make([]Vec3, 0, 20)

	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to open file:", filename)
	}
	defer file.Close()

	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "Error:", r)
		}
	}()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		switch line[0] {
		case 'v':
			parseVertex(line, &positions, &normals, &texcoords)
		case 'f':
			faces = append(faces, parseFace(line, &positions, &normals, &texcoords))
		case 'g':
			parseGroup(line)
		case 'o':
			parseObject(line)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading file:", err)
	}

	return faces
}
