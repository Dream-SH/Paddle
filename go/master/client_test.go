package master_test

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PaddlePaddle/Paddle/go/master"
	"github.com/PaddlePaddle/recordio"
)

func TestNextRecord(t *testing.T) {
	const (
		path  = "/tmp/master_client_TestFull"
		total = 50
	)

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}

	ss := strings.Split(l.Addr().String(), ":")
	p, err := strconv.Atoi(ss[len(ss)-1])
	if err != nil {
		panic(err)
	}

	go func(l net.Listener) {
		s, err := master.NewService(&master.InMemStore{}, 10, time.Second, 1)
		if err != nil {
			panic(err)
		}

		server := rpc.NewServer()
		err = server.Register(s)
		if err != nil {
			panic(err)
		}

		mux := http.NewServeMux()
		mux.Handle(rpc.DefaultRPCPath, server)
		err = http.Serve(l, mux)
		if err != nil {
			panic(err)
		}
	}(l)

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}

	w := recordio.NewWriter(f, -1, -1)
	for i := 0; i < total; i++ {
		w.Write([]byte{byte(i)})
	}
	w.Close()
	f.Close()

	c := master.NewClient(master.TestAddresser(fmt.Sprintf(":%d", p)), 10)
	c.SetDataset([]string{path})

	for pass := 0; pass < 50; pass++ {
		received := make(map[byte]bool)
		for i := 0; i <= total; i++ {
			r, err := c.NextRecord()
			if err == io.EOF {
				break
			}

			if err != nil {
				t.Fatal(pass, i, "Read error:", err)
			}

			if len(r) != 1 {
				t.Fatal(pass, i, "Length should be 1.", r)
			}

			if received[r[0]] {
				t.Fatal(pass, i, "Received duplicate.", received, r)
			}
			received[r[0]] = true
		}
	}
}
