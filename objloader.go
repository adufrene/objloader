package objloader

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Vec3 struct {
	X float32
	Y float32
	Z float32
}

type Mesh struct {
	Positions []Vec3
	Normals   []Vec3
	TexCoords []Vec3
	Indices   []uint32
}

type vertex struct {
	position Vec3
	normal   Vec3
	texCoord Vec3
}

type errScanner struct {
	scanner *bufio.Scanner
	err     error
}

type numParser struct {
	err error
}

func newMesh() Mesh {
	positions := make([]Vec3, 0, 20)
	normals := make([]Vec3, 0, 20)
	texcoords := make([]Vec3, 0, 20)
	indices := make([]uint32, 0, 20)
	return Mesh{Positions: positions, Normals: normals, TexCoords: texcoords, Indices: indices}
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

func parseVertex(line string, mesh *Mesh) {
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
		checkErr(*es, fp, errText)

		v := Vec3{X: x, Y: y, Z: z}
		mesh.Positions = append(mesh.Positions, v)
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
		mesh.TexCoords = append(mesh.TexCoords, vt)
	case "vn":
		x := fp.parseFloat(es.scan())
		y := fp.parseFloat(es.scan())
		z := fp.parseFloat(es.scan())
		checkErr(*es, fp, errText)

		vn := Vec3{X: x, Y: y, Z: z}
		mesh.Normals = append(mesh.Normals, vn)
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

func getVec3(vecs []Vec3, ndx int64) Vec3 {
	return (vecs)[fixNdx(ndx, len(vecs))]
}

func addIndices(verts []vertex, mesh *Mesh, vCache *map[vertex]uint32) {
	for i := range verts {
		v := verts[i]
		// In Cache
		if val, ok := (*vCache)[v]; ok {
			// Reuse
			mesh.Indices = append(mesh.Indices, val)
			continue
		}

		// Not in cache
		ndx := uint32(len(mesh.Positions))
		mesh.Positions = append(mesh.Positions, v.position)
		mesh.Normals = append(mesh.Normals, v.normal)
		mesh.TexCoords = append(mesh.TexCoords, v.texCoord)

		if len(mesh.Positions) == len(mesh.Normals) && len(mesh.Positions) == len(mesh.TexCoords) {
			mesh.Indices = append(mesh.Indices, ndx)
			(*vCache)[v] = ndx
		} else {
			panic("Uneven lengths of lists")
		}
	}
}

func parseFace(line string, mesh *Mesh) (verts []vertex) {
	verts = make([]vertex, 0, 3)
	errText := "Invalid face format: " + line
	s := bufio.NewScanner(strings.NewReader(line))
	s.Split(bufio.ScanWords)
	if !s.Scan() || s.Text() != "f" {
		panic(errText)
	}

	for s.Scan() {
		data := s.Text()
		s := bufio.NewScanner(strings.NewReader(data))
		s.Split(splitSlash)
		es := errScanner{scanner: s}
		ip := numParser{}
		v := vertex{}
		switch strings.Count(data, "/") {
		case 0:
			ndx, err := strconv.ParseInt(data, 10, 0)
			if err != nil {
				panic(errText)
			}
			v.position = getVec3(mesh.Positions, ndx)
		case 1:
			ndx1 := ip.parseInt(es.scan())
			ndx2 := ip.parseInt(es.scan())
			checkErr(es, ip, errText)
			v.position = getVec3(mesh.Positions, ndx1)
			v.texCoord = getVec3(mesh.TexCoords, ndx2)
		case 2:
			ndx1 := ip.parseInt(es.scan())
			v.position = getVec3(mesh.Positions, ndx1)
			if 1 == strings.Count(data, "//") {
				es.advance()
				ndx2 := ip.parseInt(es.scan())
				checkErr(es, ip, errText)
				v.normal = getVec3(mesh.Normals, ndx2)
			} else {
				ndx2 := ip.parseInt(es.scan())
				ndx3 := ip.parseInt(es.scan())
				checkErr(es, ip, errText)
				v.texCoord = getVec3(mesh.TexCoords, ndx2)
				v.normal = getVec3(mesh.Normals, ndx3)
			}
		default:
			panic(errText)
		}
		verts = append(verts, v)
	}
	return
}

func parseGroup(line string) {

}

func parseObject(line string) {

}

func LoadObj(filename string) (err error, meshes []Mesh) {
	meshes = make([]Mesh, 0, 20)
	meshes = append(meshes, newMesh())

	tempMesh := newMesh()

	currMesh := 0
	vCache := make(map[vertex]uint32)

	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to open file:", filename)
		return
	}
	defer file.Close()

	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "Error:", r)
			err = fmt.Errorf("%v", r)
		}
	}()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			// Skip empty lines
			continue
		}
		switch line[0] {
		case 'v':
			parseVertex(line, &tempMesh)
		case 'f':
			addIndices(parseFace(line, &tempMesh), &meshes[currMesh], &vCache)
		case 'g':
			parseGroup(line)
		case 'o':
			parseObject(line)
		}
	}
	if err = scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading file:", err)
	}

	return
}
