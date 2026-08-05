package matrix

import (
	// zmq "github.com/pebbe/zmq4"
	"net"
	// "time"
)

type matrix struct {
	//socket         *zmq.Socket
	//context        *zmq.Context
	udp            net.Conn
	bitmap         [][][3]byte
	flame_buffer   [][]byte
	flame_palette  [][3]byte
	rows           int
	columns        int
	plasma_palette [][3]byte
}

func Init(host string, rows int, columns int) *matrix {
	var matrix = matrix{
		rows:    rows,
		columns: columns,
	}
	/*
		context, err := zmq.NewContext()
		if err != nil {
			panic(err)
		}
		matrix.context = context
		socket, err := matrix.context.NewSocket(zmq.PUSH)
		if err != nil {
			panic(err)
		}
		matrix.socket = socket
		matrix.socket.SetLinger(1 * time.Second)
		if err := matrix.socket.Connect(host); err != nil {
			panic(err)
		}
	*/

	con, err := net.Dial("udp", host)
	if err != nil {
		panic(err)
	}
	matrix.udp = con

	matrix.bitmap = make([][][3]byte, rows)
	for r, _ := range matrix.bitmap {
		matrix.bitmap[r] = make([][3]byte, columns)
	}

	return &matrix
}

func (matrix *matrix) Send() {
	var data []byte
	for _, row := range matrix.bitmap {
		for _, b := range row {
			data = append(data, b[0], b[1], b[2])
		}
	}
	if _, err := matrix.udp.Write(data); err != nil {
		panic(err)
	}
}

func (matrix *matrix) Fill(color [3]byte) {
	for r, row := range matrix.bitmap {
		for c, _ := range row {
			matrix.bitmap[r][c] = color
		}
	}
}

func (matrix *matrix) Close() {
	matrix.Fill(ColorBlack())
	matrix.Send()
	//matrix.socket.Close()
	//matrix.context.Term()
	matrix.udp.Close()
}

func (matrix *matrix) SetPixel(r int, c int, color [3]byte) {
	if r >= matrix.rows || c >= matrix.columns || r < 0 || c < 0 {
		// Pixel out of matrix area
		return
	}
	matrix.bitmap[r][c] = color
}

func ColorBlack() [3]byte {
	return [3]byte{0, 0, 0}
}

func ColorWhite() [3]byte {
	return [3]byte{255, 255, 255}
}
